package mqtt

import (
	"os"
	"testing"

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
