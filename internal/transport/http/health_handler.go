// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: internal/transport/http/health_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-28
// Description: Automated application health checks and diagnostic telemetry endpoint.

package http

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// MQTTStatusProvider defines status introspection for the MQTT transport.
type MQTTStatusProvider interface {
	IsConnected() bool
}

// HealthStatusResponse represents the unified health status payload.
type HealthStatusResponse struct {
	Status        string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp     time.Time              `json:"timestamp"`
	UptimeSeconds int64                  `json:"uptime_seconds"`
	Components    map[string]interface{} `json:"components"`
	System        SystemMetrics          `json:"system"`
}

// SystemMetrics contains runtime memory and concurrency metrics.
type SystemMetrics struct {
	Goroutines   int    `json:"goroutines"`
	HeapAllocMB  uint64 `json:"heap_alloc_mb"`
	HeapSysMB    uint64 `json:"heap_sys_mb"`
	NumGC        uint32 `json:"num_gc"`
	GoVersion    string `json:"go_version"`
}

// HealthHandler manages application health and liveness probes.
type HealthHandler struct {
	dbService  DatabaseService
	mqttStatus MQTTStatusProvider
	deviceRepo domain.DeviceRepository
	startTime  time.Time
}

// NewHealthHandler creates a new HealthHandler instance.
func NewHealthHandler(dbService DatabaseService, mqttStatus MQTTStatusProvider, dRepo domain.DeviceRepository) *HealthHandler {
	return &HealthHandler{
		dbService:  dbService,
		mqttStatus: mqttStatus,
		deviceRepo: dRepo,
		startTime:  time.Now(),
	}
}

// ServeLiveness responds to kubernetes/systemd liveness probes with HTTP 200.
func (h *HealthHandler) ServeLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK\n"))
}

// ServeHealth provides detailed component status and diagnostics in JSON format.
func (h *HealthHandler) ServeHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	uptime := int64(now.Sub(h.startTime).Seconds())

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	sysMetrics := SystemMetrics{
		Goroutines:  runtime.NumGoroutine(),
		HeapAllocMB: mem.HeapAlloc / (1024 * 1024),
		HeapSysMB:   mem.HeapSys / (1024 * 1024),
		NumGC:       mem.NumGC,
		GoVersion:   runtime.Version(),
	}

	components := make(map[string]interface{})
	overallStatus := "healthy"

	// 1. Database check
	if h.dbService != nil {
		dbStat := h.dbService.GetStatus()
		components["database"] = map[string]interface{}{
			"status":     map[bool]string{true: "up", false: "down"}[dbStat.Connected],
			"type":       dbStat.Type,
			"latency_ms": dbStat.LatencyMs,
			"error":      dbStat.ErrorMessage,
		}
		if !dbStat.Connected {
			overallStatus = "unhealthy"
		}
	} else {
		components["database"] = map[string]interface{}{
			"status": "unconfigured",
		}
	}

	// 2. MQTT check
	mqttConnected := false
	if h.mqttStatus != nil {
		mqttConnected = h.mqttStatus.IsConnected()
	}
	components["mqtt"] = map[string]interface{}{
		"connected": mqttConnected,
		"status":    map[bool]string{true: "up", false: "disconnected"}[mqttConnected],
	}
	if !mqttConnected && overallStatus != "unhealthy" {
		overallStatus = "degraded"
	}

	// 3. Registered Devices
	deviceCount := 0
	if h.deviceRepo != nil {
		if devices, err := h.deviceRepo.GetAllDevices(); err == nil {
			deviceCount = len(devices)
		}
	}
	components["devices"] = map[string]interface{}{
		"count": deviceCount,
	}

	response := HealthStatusResponse{
		Status:        overallStatus,
		Timestamp:     now.UTC(),
		UptimeSeconds: uptime,
		Components:    components,
		System:        sysMetrics,
	}

	if overallStatus == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(response)
}
