// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/tunnel/duckdns_driver.go
// Author: Gabriel Moraes
// Date: 2026-09-13
// Description: Pure Go DDNS provider implementing tunnel.Driver for DuckDNS.

package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultDuckDNSRefreshInterval = 10 * time.Minute
	duckDNSUpdateEndpoint         = "https://www.duckdns.org/update"
)

// DuckDNSDriver handles periodic registration of IPv4/IPv6 with DuckDNS.
// Implements tunnel.Driver and IPv6Reporter interfaces.
type DuckDNSDriver struct {
	mu              sync.RWMutex
	httpClient      *http.Client
	updateURL       string
	refreshInterval time.Duration

	subdomain   string
	token       string
	port        string
	publicURL   string
	ipv6Address string
	useHTTPS    bool

	cancel context.CancelFunc
	doneCh chan struct{}
}

// NewDuckDNSDriver creates a new DuckDNSDriver with default settings.
func NewDuckDNSDriver() *DuckDNSDriver {
	useHTTPS := os.Getenv("MONITOR_USE_HTTPS") == "true" || os.Getenv("DUCKDNS_USE_HTTPS") == "true"
	return &DuckDNSDriver{
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		updateURL:       duckDNSUpdateEndpoint,
		refreshInterval: defaultDuckDNSRefreshInterval,
		ipv6Address:     DetectGlobalIPv6(),
		useHTTPS:        useHTTPS,
	}
}

// SetHTTPS explicitly enables or disables HTTPS public URL formatting.
func (d *DuckDNSDriver) SetHTTPS(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.useHTTPS = enabled
}

// IsHTTPS reports whether DuckDNS is configured for HTTPS reverse proxy.
func (d *DuckDNSDriver) IsHTTPS() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.useHTTPS
}

// Name returns the provider identifier.
func (d *DuckDNSDriver) Name() string {
	return "duckdns"
}

// IsAvailable always returns true because DuckDNS uses pure Go HTTP and requires no external CLI.
func (d *DuckDNSDriver) IsAvailable() bool {
	return true
}

// GetIPv6Address returns the cached global IPv6 detected on the host or inspects interfaces on demand.
func (d *DuckDNSDriver) GetIPv6Address() string {
	d.mu.RLock()
	cached := d.ipv6Address
	d.mu.RUnlock()

	if cached != "" {
		return cached
	}
	detected := DetectGlobalIPv6()
	if detected != "" {
		d.mu.Lock()
		d.ipv6Address = detected
		d.mu.Unlock()
	}
	return detected
}

// Start launches the DuckDNS periodic updater for the provided configuration.
func (d *DuckDNSDriver) Start(ctx context.Context, cfg Config) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cleanToken := CleanDuckDNSToken(cfg.AuthToken)
	if cleanToken == "" {
		return fmt.Errorf("duckdns token is required")
	}

	subdomain := CleanDuckDNSDomain(cfg.Domain)
	if subdomain == "" {
		return fmt.Errorf("duckdns subdomain is required")
	}
	if strings.Contains(subdomain, "ngrok") {
		return fmt.Errorf("o domínio informado pertence ao Ngrok ('%s'). Por favor, insira o seu subdomínio do DuckDNS criado em duckdns.org", subdomain)
	}

	port := cfg.LocalPort
	if port == "" {
		port = "22100"
	}

	// Terminate any previous execution
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}

	procCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.doneCh = make(chan struct{})
	d.token = cleanToken
	d.subdomain = subdomain
	d.port = port
	d.publicURL = ""

	// 1. Detect IPv6
	ipv6 := DetectGlobalIPv6()
	d.ipv6Address = ipv6
	if ipv6 != "" {
		log.Printf("[DUCKDNS] Detected public global IPv6: %s", ipv6)
	} else {
		log.Printf("[DUCKDNS] No public global IPv6 found. Operating in IPv4 mode.")
	}

	// 2. Perform initial synchronous update
	if err := d.executeUpdate(procCtx, subdomain, cleanToken, ipv6); err != nil {
		cancel()
		close(d.doneCh)
		return fmt.Errorf("failed initial duckdns update: %w", err)
	}

	if cfg.UseHTTPS || os.Getenv("MONITOR_USE_HTTPS") == "true" || os.Getenv("DUCKDNS_USE_HTTPS") == "true" {
		d.useHTTPS = true
	}

	if d.useHTTPS {
		d.publicURL = fmt.Sprintf("https://%s.duckdns.org", subdomain)
	} else {
		d.publicURL = fmt.Sprintf("http://%s.duckdns.org:%s", subdomain, port)
	}
	log.Printf("[DUCKDNS] Successfully registered domain '%s.duckdns.org' -> %s", subdomain, d.publicURL)

	// 3. Launch periodic updater loop
	go d.loop(procCtx, subdomain, cleanToken)

	return nil
}

// executeUpdate calls the DuckDNS API to update domain records.
func (d *DuckDNSDriver) executeUpdate(ctx context.Context, subdomain, token, ipv6 string) error {
	params := url.Values{}
	params.Set("domains", subdomain)
	params.Set("token", token)
	params.Set("ip", "") // Leave empty so DuckDNS detects public IPv4 automatically
	if ipv6 != "" {
		params.Set("ipv6", ipv6)
	}

	reqURL := fmt.Sprintf("%s?%s", d.updateURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create duckdns request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading duckdns response: %w", err)
	}

	body := strings.TrimSpace(string(bodyBytes))
	if !strings.EqualFold(body, "OK") {
		return fmt.Errorf("duckdns rejected update (response: '%s'). Check token and domain", body)
	}

	return nil
}

// loop runs the periodic heartbeat to keep the DuckDNS domain mapped.
func (d *DuckDNSDriver) loop(ctx context.Context, subdomain, token string) {
	ticker := time.NewTicker(d.refreshInterval)
	defer func() {
		ticker.Stop()
		close(d.doneCh)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("[DUCKDNS] Updater stopped.")
			return
		case <-ticker.C:
			ipv6 := DetectGlobalIPv6()
			d.mu.Lock()
			d.ipv6Address = ipv6
			d.mu.Unlock()

			if err := d.executeUpdate(ctx, subdomain, token, ipv6); err != nil {
				log.Printf("[DUCKDNS] Periodic update warning: %v", err)
			} else {
				log.Printf("[DUCKDNS] Periodic heartbeat refreshed successfully.")
			}
		}
	}
}

// Stop terminates the periodic background updater.
func (d *DuckDNSDriver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.publicURL = ""
	return nil
}

// Wait blocks until the driver's background routine has fully exited.
func (d *DuckDNSDriver) Wait() error {
	d.mu.RLock()
	done := d.doneCh
	d.mu.RUnlock()

	if done == nil {
		return nil
	}

	<-done
	return nil
}

// GetPublicURL returns the live DuckDNS public address.
func (d *DuckDNSDriver) GetPublicURL(ctx context.Context) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.publicURL == "" {
		return "", fmt.Errorf("duckdns service not started")
	}

	return d.publicURL, nil
}
