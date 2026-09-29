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
// Date: 2026-01-19

package mqtt

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/protocol"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps the Paho MQTT client to provide specific Noxfort functionality.
// It bridges the gap between raw MQTT messages and the Domain Logic.
type Client struct {
	internalClient mqtt.Client
	stateManager   monitor.EventProcessor
	topicPattern   string
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
	if token := c.internalClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %w", token.Error())
	}

	// Subscribe using the wildcard pattern
	if token := c.internalClient.Subscribe(c.topicPattern, 1, c.handleMessage); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", c.topicPattern, token.Error())
	}

	log.Printf("[MQTT] Subscribed to %s (Listening for JSON events)", c.topicPattern)
	return nil
}

// handleMessage is the pipeline entry point: Raw JSON -> DecodePayload -> State Manager.
func (c *Client) handleMessage(client mqtt.Client, msg mqtt.Message) {
	payload := msg.Payload()

	// Decode and validate using the unified protocol decoder (Single Responsibility / DRY)
	event, err := protocol.DecodePayload(payload)
	if err != nil {
		log.Printf("[MQTT] Decode Error on topic %s: %v", msg.Topic(), err)
		return
	}

	// Pass the validated event to the State Manager
	c.stateManager.ProcessEvent(event.Origin, event)
}

// Disconnect gracefully closes the connection.
func (c *Client) Disconnect() {
	c.internalClient.Disconnect(250)
	log.Println("[MQTT] Disconnected.")
}

// IsConnected returns whether the internal MQTT client is currently connected.
func (c *Client) IsConnected() bool {
	if c == nil || c.internalClient == nil {
		return false
	}
	return c.internalClient.IsConnected()
}
