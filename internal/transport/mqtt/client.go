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
// File: internal/transport/mqtt/client.go
// Author: Gabriel Moraes
// Date: October 2026

package mqtt

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/metrics"
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/protocol"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DevicePolicyChecker validates whether an MQTT origin is registered, ACTIVE, and enabled.
type DevicePolicyChecker interface {
	GetByIdentifier(identifier string) (*domain.Device, error)
}

// Client wraps the Paho MQTT client to provide specific Noxfort functionality.
// It bridges the gap between raw MQTT messages and the Domain Logic.
type Client struct {
	internalClient mqtt.Client
	stateManager   monitor.EventProcessor
	topicPattern   string
	policyMu       sync.RWMutex
	devicePolicy   DevicePolicyChecker
	strictZeroTrust bool
}

// SetDevicePolicy configures device registration and Maintenance Mode enforcement for incoming MQTT telemetry.
func (c *Client) SetDevicePolicy(checker DevicePolicyChecker, strictZeroTrust bool) {
	c.policyMu.Lock()
	defer c.policyMu.Unlock()
	c.devicePolicy = checker
	c.strictZeroTrust = strictZeroTrust
}

func (c *Client) getDevicePolicy() (DevicePolicyChecker, bool) {
	c.policyMu.RLock()
	defer c.policyMu.RUnlock()
	return c.devicePolicy, c.strictZeroTrust
}

// NewClient creates a configured MQTT client instance using environment variables
// (MQTT_USER / MQTT_PASSWORD) or URL credentials if available.
func NewClient(brokerURL string, sm monitor.EventProcessor) *Client {
	username := os.Getenv("MQTT_USER")
	password := os.Getenv("MQTT_PASSWORD")
	return NewClientWithCredentials(brokerURL, username, password, sm)
}

// NewClientWithCredentials creates a configured MQTT client with explicit credentials.
func NewClientWithCredentials(brokerURL string, username, password string, sm monitor.EventProcessor) *Client {
	cleanBrokerURL := brokerURL
	// Parse credentials embedded in URL if parameters are empty
	if parsedURL, err := url.Parse(brokerURL); err == nil {
		if username == "" && parsedURL.User != nil {
			username = parsedURL.User.Username()
			if p, ok := parsedURL.User.Password(); ok {
				password = p
			}
			// Strip user info from the broker URL for Paho
			parsedURL.User = nil
			cleanBrokerURL = parsedURL.String()
		}
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cleanBrokerURL)
	opts.SetClientID("noxfort-monitor-server")
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)

	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("[MQTT] Connection established.")
	})

	c := mqtt.NewClient(opts)

	return &Client{
		internalClient: c,
		stateManager:   sm,
		// Wildcard subscription: Catch EVERYTHING under noxfort/telemetry/
		// We rely on the JSON "origin" field to identify the source.
		topicPattern: "noxfort/telemetry/#",
	}
}

// Connect establishes the connection and subscribes to the telemetry topic.
func (c *Client) Connect() error {
	c.policyMu.Lock()
	client := c.internalClient
	c.policyMu.Unlock()

	if client == nil {
		return fmt.Errorf("mqtt client not initialized")
	}
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %w", token.Error())
	}

	// Subscribe using the wildcard pattern
	if token := client.Subscribe(c.topicPattern, 1, c.handleMessage); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.topicPattern, token.Error())
	}

	log.Printf("[MQTT] Subscribed to %s (Listening for JSON events)", c.topicPattern)
	return nil
}

// Reconnect swaps the broker endpoint dynamically without restarting the application.
func (c *Client) Reconnect(brokerURL string) error {
	username := os.Getenv("MQTT_USER")
	password := os.Getenv("MQTT_PASSWORD")
	fresh := NewClientWithCredentials(brokerURL, username, password, c.stateManager)

	c.policyMu.Lock()
	old := c.internalClient
	c.internalClient = fresh.internalClient
	c.policyMu.Unlock()

	if old != nil && old.IsConnected() {
		old.Disconnect(250)
	}
	return c.Connect()
}

// handleMessage is the pipeline entry point: Raw JSON -> DecodePayload -> Zero-Trust Check -> State Manager.
func (c *Client) handleMessage(client mqtt.Client, msg mqtt.Message) {
	payload := msg.Payload()

	// Decode and validate using the unified protocol decoder (Single Responsibility / DRY)
	event, err := protocol.DecodePayload(payload)
	if err != nil {
		metrics.IncTelemetryRejected("mqtt", "decode")
		log.Printf("[MQTT] Decode Error on topic %s: %v", msg.Topic(), err)
		return
	}

	if checker, strict := c.getDevicePolicy(); checker != nil {
		dev, err := checker.GetByIdentifier(event.Origin)
		if err != nil {
			metrics.IncTelemetryRejected("mqtt", "lookup_error")
			log.Printf("[MQTT-SECURITY] Failed device lookup for origin '%s': %v", event.Origin, err)
			return
		}
		if strict {
			if dev == nil {
				metrics.IncTelemetryRejected("mqtt", "unregistered")
				log.Printf("[MQTT-SECURITY] Rejected unregistered MQTT origin '%s' (Zero-Trust strict mode).", event.Origin)
				return
			}
			if dev.Status != "ACTIVE" {
				metrics.IncTelemetryRejected("mqtt", "pending")
				log.Printf("[MQTT-SECURITY] Rejected non-ACTIVE MQTT origin '%s' (status=%s).", event.Origin, dev.Status)
				return
			}
		}
		if dev != nil && !dev.Enabled {
			metrics.IncTelemetryRejected("mqtt", "paused")
			log.Printf("[MQTT] Ignored event from paused device '%s' (Maintenance Mode).", event.Origin)
			return
		}
	}

	metrics.IncTelemetryIngested("mqtt", string(event.Level))

	// Pass the validated event to the State Manager
	c.stateManager.ProcessEvent(event.Origin, event)
}

// Disconnect gracefully closes the connection.
func (c *Client) Disconnect() {
	c.policyMu.RLock()
	client := c.internalClient
	c.policyMu.RUnlock()
	if client != nil {
		client.Disconnect(250)
	}
	log.Println("[MQTT] Disconnected.")
}

// IsConnected returns whether the internal MQTT client is currently connected.
func (c *Client) IsConnected() bool {
	if c == nil {
		return false
	}
	c.policyMu.RLock()
	client := c.internalClient
	c.policyMu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsConnected()
}
