// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: internal/transport/http/metrics_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-28
// Description: Prometheus-compatible metrics exporter endpoint (/metrics).

package http

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"noxfort-monitor-server/internal/domain"
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
	if h.deviceRepo != nil {
		if devices, err := h.deviceRepo.GetAllDevices(); err == nil {
			deviceCount = len(devices)
		}
	}

	out := fmt.Sprintf(`# HELP noxfort_uptime_seconds Application uptime in seconds.
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
	)

	_, _ = w.Write([]byte(out))
}
