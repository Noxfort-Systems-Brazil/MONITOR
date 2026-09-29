// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Noxfort Systems
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
// File: internal/tunnel/duckdns_driver_test.go
// Author: Gabriel Moraes
// Date: 2026-09-13
// Description: Unit tests for DuckDNSDriver.

package tunnel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCleanDuckDNSToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  a1b2c3d4-e5f6  ", "a1b2c3d4-e5f6"},
		{`"a1b2c3d4-e5f6"`, "a1b2c3d4-e5f6"},
		{"'a1b2c3d4-e5f6'", "a1b2c3d4-e5f6"},
		{"", ""},
	}

	for _, tt := range tests {
		got := CleanDuckDNSToken(tt.input)
		if got != tt.expected {
			t.Errorf("CleanDuckDNSToken(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCleanDuckDNSDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-subdomain", "my-subdomain"},
		{"my-subdomain.duckdns.org", "my-subdomain"},
		{"https://my-subdomain.duckdns.org/", "my-subdomain"},
		{"http://my-subdomain", "my-subdomain"},
		{"  noxfort-lab.duckdns.org  ", "noxfort-lab"},
	}

	for _, tt := range tests {
		got := CleanDuckDNSDomain(tt.input)
		if got != tt.expected {
			t.Errorf("CleanDuckDNSDomain(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDuckDNSDriver_Lifecycle(t *testing.T) {
	var receivedDomains, receivedToken, receivedIPv6 string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		receivedDomains = q.Get("domains")
		receivedToken = q.Get("token")
		receivedIPv6 = q.Get("ipv6")
		_ = receivedIPv6

		if receivedToken == "valid-token" && receivedDomains == "test-domain" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("KO"))
		}
	}))
	defer server.Close()

	driver := NewDuckDNSDriver()
	driver.updateURL = server.URL
	driver.refreshInterval = 50 * time.Millisecond

	if !driver.IsAvailable() {
		t.Errorf("DuckDNSDriver should always be available")
	}

	if driver.Name() != "duckdns" {
		t.Errorf("Expected driver name 'duckdns', got %s", driver.Name())
	}

	if DetectGlobalIPv6() != "" && driver.GetIPv6Address() == "" {
		t.Errorf("Expected driver to discover global IPv6 before Start, got empty string")
	}

	ctx := context.Background()

	// 1. Test error with empty token
	err := driver.Start(ctx, Config{AuthToken: "", Domain: "test-domain"})
	if err == nil {
		t.Errorf("Expected error with empty token, got nil")
	}

	// 2. Test error with invalid token (server returns KO)
	err = driver.Start(ctx, Config{AuthToken: "invalid-token", Domain: "test-domain"})
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("Expected rejected error with invalid token, got %v", err)
	}

	// 3. Test successful start
	err = driver.Start(ctx, Config{AuthToken: "valid-token", Domain: "test-domain.duckdns.org", LocalPort: "22100"})
	if err != nil {
		t.Fatalf("Expected successful start, got %v", err)
	}

	if receivedDomains != "test-domain" {
		t.Errorf("Expected domains 'test-domain', got %s", receivedDomains)
	}
	if receivedToken != "valid-token" {
		t.Errorf("Expected token 'valid-token', got %s", receivedToken)
	}

	url, err := driver.GetPublicURL(ctx)
	if err != nil {
		t.Errorf("Expected no error getting public URL, got %v", err)
	}
	expectedURL := "http://test-domain.duckdns.org:22100"
	if url != expectedURL {
		t.Errorf("Expected public URL %q, got %q", expectedURL, url)
	}

	// 4. Test stop
	err = driver.Stop()
	if err != nil {
		t.Errorf("Expected no error stopping driver, got %v", err)
	}

	_, err = driver.GetPublicURL(ctx)
	if err == nil {
		t.Errorf("Expected error after stop, got nil")
	}
}

func TestDuckDNSDriver_Test(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("token") == "valid-tok" && q.Get("domains") == "valid-dom" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("KO"))
		}
	}))
	defer server.Close()

	driver := NewDuckDNSDriver()
	driver.updateURL = server.URL

	ctx := context.Background()

	// 1. Missing token
	_, err := driver.Test(ctx, "", "valid-dom")
	if err == nil {
		t.Errorf("Expected error with empty token")
	}

	// 2. Ngrok domain rejected
	_, err = driver.Test(ctx, "valid-tok", "my-app.ngrok-free.app")
	if err == nil || !strings.Contains(err.Error(), "Ngrok") {
		t.Errorf("Expected error for ngrok domain, got %v", err)
	}

	// 3. Rejected by DuckDNS (KO)
	_, err = driver.Test(ctx, "wrong-tok", "valid-dom")
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("Expected rejected error, got %v", err)
	}

	// 4. Successful test
	res, err := driver.Test(ctx, "valid-tok", "valid-dom.duckdns.org")
	if err != nil {
		t.Fatalf("Expected successful test, got %v", err)
	}
	if !res.Success {
		t.Errorf("Expected res.Success = true")
	}
	if res.Domain != "valid-dom.duckdns.org" {
		t.Errorf("Expected domain 'valid-dom.duckdns.org', got %s", res.Domain)
	}
}

func TestDuckDNSDriver_HTTPS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctx := context.Background()

	// Test with Config.UseHTTPS = true
	driver := NewDuckDNSDriver()
	driver.updateURL = server.URL

	err := driver.Start(ctx, Config{
		AuthToken: "token-123",
		Domain:    "secure-monitor.duckdns.org",
		LocalPort: "22100",
		UseHTTPS:  true,
	})
	if err != nil {
		t.Fatalf("Failed to start driver with HTTPS: %v", err)
	}
	defer driver.Stop()

	url, err := driver.GetPublicURL(ctx)
	if err != nil {
		t.Fatalf("Failed to get public URL: %v", err)
	}
	expected := "https://secure-monitor.duckdns.org"
	if url != expected {
		t.Errorf("Expected HTTPS public URL %q, got %q", expected, url)
	}

	if !driver.IsHTTPS() {
		t.Errorf("Expected driver.IsHTTPS() == true")
	}
}

