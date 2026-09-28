// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

func TestMetricsHandler_ServeMetrics(t *testing.T) {
	dbMock := &mockDBService{
		status: domain.DatabaseStatus{
			Connected: true,
			Type:      "sqlite",
			LatencyMs: 3,
		},
	}
	mqttMock := &mockMQTTStatus{connected: true}

	handler := NewMetricsHandler(dbMock, mqttMock, nil)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	body := rec.Body.String()

	expectedMetrics := []string{
		"noxfort_uptime_seconds",
		"noxfort_goroutines",
		"noxfort_memory_heap_alloc_bytes",
		"noxfort_db_status{type=\"sqlite\"} 1",
		"noxfort_db_latency_ms 3",
		"noxfort_mqtt_connected 1",
		"noxfort_devices_total 0",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metrics output to contain '%s', but it was missing", metric)
		}
	}
}
