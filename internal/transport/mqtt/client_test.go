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
// File: internal/transport/mqtt/client_test.go
// Author: Gabriel Moraes
// Date: October 2026

package mqtt

import (
	"os"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"noxfort-monitor-server/internal/domain"
)

type mockProcessor struct {
	events []struct {
		id    string
		event *domain.IncomingEvent
	}
}

func (m *mockProcessor) ProcessEvent(id string, event *domain.IncomingEvent) {
	m.events = append(m.events, struct {
		id    string
		event *domain.IncomingEvent
	}{id: id, event: event})
}

func TestNewClient_Options(t *testing.T) {
	proc := &mockProcessor{}

	// 1. URL with embedded credentials
	client := NewClientWithCredentials("tcp://user:secret@127.0.0.1:1883", "", "", proc)
	if client == nil {
		t.Fatal("Expected non-nil MQTT client")
	}
	if client.topicPattern != "noxfort/telemetry/#" {
		t.Errorf("Expected topic 'noxfort/telemetry/#', got %s", client.topicPattern)
	}

	// 2. Explicit credentials
	client2 := NewClientWithCredentials("tcp://127.0.0.1:1883", "admin", "pass123", proc)
	if client2 == nil {
		t.Fatal("Expected non-nil MQTT client")
	}

	// 3. Environment variables fallback
	os.Setenv("MQTT_USER", "env_user")
	os.Setenv("MQTT_PASSWORD", "env_pass")
	defer func() {
		os.Unsetenv("MQTT_USER")
		os.Unsetenv("MQTT_PASSWORD")
	}()

	client3 := NewClient("tcp://127.0.0.1:1883", proc)
	if client3 == nil {
		t.Fatal("Expected non-nil MQTT client from env")
	}

	// 4. IsConnected when not connected
	if client.IsConnected() {
		t.Error("Expected IsConnected to be false before Connect()")
	}
}

type mockMQTTDevicePolicy struct {
	devices map[string]*domain.Device
}

func (m *mockMQTTDevicePolicy) GetByIdentifier(identifier string) (*domain.Device, error) {
	if dev, ok := m.devices[identifier]; ok {
		return dev, nil
	}
	return nil, nil
}

type fakeMQTTMsg struct {
	topic   string
	payload []byte
}

func (f fakeMQTTMsg) Duplicate() bool   { return false }
func (f fakeMQTTMsg) Qos() byte         { return 1 }
func (f fakeMQTTMsg) Retained() bool    { return false }
func (f fakeMQTTMsg) Topic() string     { return f.topic }
func (f fakeMQTTMsg) MessageID() uint16 { return 1 }
func (f fakeMQTTMsg) Payload() []byte   { return f.payload }
func (f fakeMQTTMsg) Ack()              {}

var _ mqtt.Message = fakeMQTTMsg{}

func TestMQTTClient_ZeroTrustPolicyEnforcement(t *testing.T) {
	proc := &mockProcessor{}
	policy := &mockMQTTDevicePolicy{
		devices: map[string]*domain.Device{
			"active-node":  {Identifier: "active-node", Status: "ACTIVE", Enabled: true},
			"pending-node": {Identifier: "pending-node", Status: "PENDING", Enabled: true},
			"paused-node":  {Identifier: "paused-node", Status: "ACTIVE", Enabled: false},
		},
	}

	client := NewClientWithCredentials("tcp://127.0.0.1:1883", "", "", proc)
	client.SetDevicePolicy(policy, true)

	// 1. Unregistered origin blocked
	client.handleMessage(nil, fakeMQTTMsg{
		topic:   "noxfort/telemetry/unknown",
		payload: []byte(`{"origin":"rogue-node","level":"CRITICAL","message":"Injected","category":"SOFTWARE"}`),
	})
	// 2. PENDING device blocked
	client.handleMessage(nil, fakeMQTTMsg{
		topic:   "noxfort/telemetry/pending",
		payload: []byte(`{"origin":"pending-node","level":"WARNING","message":"Pending","category":"HARDWARE"}`),
	})
	// 3. Paused device ignored (Maintenance Mode)
	client.handleMessage(nil, fakeMQTTMsg{
		topic:   "noxfort/telemetry/paused",
		payload: []byte(`{"origin":"paused-node","level":"CRITICAL","message":"Paused","category":"HARDWARE"}`),
	})
	if len(proc.events) != 0 {
		t.Fatalf("Expected 0 processed events for rogue/pending/paused MQTT sources, got %d", len(proc.events))
	}

	// 4. ACTIVE & Enabled device allowed
	client.handleMessage(nil, fakeMQTTMsg{
		topic:   "noxfort/telemetry/active",
		payload: []byte(`{"origin":"active-node","level":"CRITICAL","message":"Real failure","category":"HARDWARE"}`),
	})
	if len(proc.events) != 1 || proc.events[0].id != "active-node" {
		t.Fatalf("Expected 1 processed event for active-node, got %+v", proc.events)
	}
}
