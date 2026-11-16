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
// File: internal/transport/http/metrics_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/metrics"
)

// MetricsHandler exports Prometheus exposition format telemetry.
type MetricsHandler struct {
	dbService  DatabaseService
	mqttStatus MQTTStatusProvider
	deviceRepo domain.DeviceRepository
	startTime  time.Time
}

// NewMetricsHandler creates a new Prometheus metrics exporter handler.
func NewMetricsHandler(dbService DatabaseService, mqttStatus MQTTStatusProvider, dRepo domain.DeviceRepository) *MetricsHandler {
	return &MetricsHandler{
		dbService:  dbService,
		mqttStatus: mqttStatus,
		deviceRepo: dRepo,
		startTime:  time.Now(),
	}
}

// ServeMetrics serves application metrics in Prometheus text format (version 0.0.4).
func (h *MetricsHandler) ServeMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	uptime := int64(time.Since(h.startTime).Seconds())

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	goroutines := runtime.NumGoroutine()

	dbConnected := 0
	dbLatency := int64(0)
	dbType := "none"
	if h.dbService != nil {
		stat := h.dbService.GetStatus()
		if stat.Connected {
			dbConnected = 1
		}
		dbLatency = stat.LatencyMs
		if stat.Type != "" {
			dbType = stat.Type
		}
	}

	mqttConnected := 0
	if h.mqttStatus != nil && h.mqttStatus.IsConnected() {
		mqttConnected = 1
	}

	deviceCount := 0
	statusCounts := map[string]int{
		"online":  0,
		"offline": 0,
		"pending": 0,
		"paused":  0,
		"waiting": 0,
	}
	if h.deviceRepo != nil {
		if devices, err := h.deviceRepo.GetAllDevices(); err == nil {
			deviceCount = len(devices)
			threshold := 5 * time.Minute
			for _, d := range devices {
				status, _, _, _ := classifyDeviceStatus(d, threshold)
				statusCounts[strings.ToLower(status)]++
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `# HELP noxfort_uptime_seconds Application uptime in seconds.
# TYPE noxfort_uptime_seconds gauge
noxfort_uptime_seconds %d

# HELP noxfort_goroutines Current number of running goroutines.
# TYPE noxfort_goroutines gauge
noxfort_goroutines %d

# HELP noxfort_memory_heap_alloc_bytes Bytes of allocated heap objects.
# TYPE noxfort_memory_heap_alloc_bytes gauge
noxfort_memory_heap_alloc_bytes %d

# HELP noxfort_memory_heap_sys_bytes Bytes of heap memory obtained from OS.
# TYPE noxfort_memory_heap_sys_bytes gauge
noxfort_memory_heap_sys_bytes %d

# HELP noxfort_memory_num_gc Total number of completed GC cycles.
# TYPE noxfort_memory_num_gc counter
noxfort_memory_num_gc %d

# HELP noxfort_db_status Active database connection status (1 = connected, 0 = disconnected).
# TYPE noxfort_db_status gauge
noxfort_db_status{type="%s"} %d

# HELP noxfort_db_latency_ms Active database ping latency in milliseconds.
# TYPE noxfort_db_latency_ms gauge
noxfort_db_latency_ms %d

# HELP noxfort_mqtt_connected MQTT broker connection status (1 = connected, 0 = disconnected).
# TYPE noxfort_mqtt_connected gauge
noxfort_mqtt_connected %d

# HELP noxfort_devices_total Total number of registered industrial devices.
# TYPE noxfort_devices_total gauge
noxfort_devices_total %d

# HELP noxfort_devices_by_status Registered devices broken down by operational state.
# TYPE noxfort_devices_by_status gauge
noxfort_devices_by_status{state="online"} %d
noxfort_devices_by_status{state="offline"} %d
noxfort_devices_by_status{state="pending"} %d
noxfort_devices_by_status{state="paused"} %d
noxfort_devices_by_status{state="waiting"} %d

# HELP noxfort_telemetry_buffer_depth Current number of queued telemetry records waiting for batch flush.
# TYPE noxfort_telemetry_buffer_depth gauge
noxfort_telemetry_buffer_depth %d
`,
		uptime,
		goroutines,
		mem.HeapAlloc,
		mem.HeapSys,
		mem.NumGC,
		dbType, dbConnected,
		dbLatency,
		mqttConnected,
		deviceCount,
		statusCounts["online"],
		statusCounts["offline"],
		statusCounts["pending"],
		statusCounts["paused"],
		statusCounts["waiting"],
		metrics.GetBufferDepth(),
	)

	ingested := metrics.SnapshotIngested()
	if len(ingested) > 0 {
		b.WriteString("\n# HELP noxfort_telemetry_ingested_total Accepted telemetry messages by transport and level.\n# TYPE noxfort_telemetry_ingested_total counter\n")
		for _, entry := range ingested {
			fmt.Fprintf(&b, "noxfort_telemetry_ingested_total{transport=%q,level=%q} %d\n", entry.LabelA, entry.LabelB, entry.Value)
		}
	}

	rejected := metrics.SnapshotRejected()
	if len(rejected) > 0 {
		b.WriteString("\n# HELP noxfort_telemetry_rejected_total Rejected telemetry messages by transport and reason.\n# TYPE noxfort_telemetry_rejected_total counter\n")
		for _, entry := range rejected {
			fmt.Fprintf(&b, "noxfort_telemetry_rejected_total{transport=%q,reason=%q} %d\n", entry.LabelA, entry.LabelB, entry.Value)
		}
	}

	dispatched := metrics.SnapshotAlerts()
	if len(dispatched) > 0 {
		b.WriteString("\n# HELP noxfort_alerts_dispatched_total Alert delivery attempts by channel and status.\n# TYPE noxfort_alerts_dispatched_total counter\n")
		for _, entry := range dispatched {
			fmt.Fprintf(&b, "noxfort_alerts_dispatched_total{channel=%q,status=%q} %d\n", entry.LabelA, entry.LabelB, entry.Value)
		}
	}

	_, _ = w.Write([]byte(b.String()))
}
