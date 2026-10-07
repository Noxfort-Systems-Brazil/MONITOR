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
// File: internal/transport/http/metrics_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-28

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
