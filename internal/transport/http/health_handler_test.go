// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

type mockMQTTStatus struct {
	connected bool
}

func (m *mockMQTTStatus) IsConnected() bool {
	return m.connected
}

func TestHealthHandler_ServeLiveness(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeLiveness(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}
	if rec.Body.String() != "OK\n" {
		t.Fatalf("expected 'OK\\n', got: %s", rec.Body.String())
	}
}

func TestHealthHandler_ServeHealth(t *testing.T) {
	dbMock := &mockDBService{
		status: domain.DatabaseStatus{
			Connected: true,
			Type:      "sqlite",
			LatencyMs: 2,
		},
	}
	mqttMock := &mockMQTTStatus{connected: true}

	handler := NewHealthHandler(dbMock, mqttMock, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var res HealthStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if res.Status != "healthy" {
		t.Errorf("expected status 'healthy', got: %s", res.Status)
	}
	if res.System.Goroutines <= 0 {
		t.Errorf("expected goroutines > 0, got: %d", res.System.Goroutines)
	}
}
