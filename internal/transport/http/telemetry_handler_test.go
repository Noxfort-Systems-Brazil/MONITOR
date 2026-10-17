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
// File: internal/transport/http/telemetry_handler_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/storage"
)

type mockTelemetryProcessor struct {
	mu     sync.Mutex
	events []struct {
		identifier string
		event      *domain.IncomingEvent
	}
}

func (m *mockTelemetryProcessor) ProcessEvent(identifier string, event *domain.IncomingEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, struct {
		identifier string
		event      *domain.IncomingEvent
	}{identifier: identifier, event: event})
}

func TestTelemetryHandler_HandleIngest(t *testing.T) {
	processor := &mockTelemetryProcessor{}
	handler := NewTelemetryHandler(processor, nil)

	// 1. Method Not Allowed (GET)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	rrGet := httptest.NewRecorder()
	handler.HandleIngest(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 on GET /api/telemetry, got %d", rrGet.Code)
	}

	// 2. Invalid JSON payload
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader("{invalid-json"))
	rrBadJSON := httptest.NewRecorder()
	handler.HandleIngest(rrBadJSON, reqBadJSON)
	if rrBadJSON.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 on invalid JSON, got %d", rrBadJSON.Code)
	}

	// 3. Valid JSON telemetry payload
	validJSON := `{
		"origin": "SENSOR-PRESSURE-01",
		"level": "WARNING",
		"category": "HARDWARE",
		"message": "High pipeline pressure detected"
	}`
	reqValid := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader(validJSON))
	rrValid := httptest.NewRecorder()
	handler.HandleIngest(rrValid, reqValid)

	if rrValid.Code != http.StatusOK {
		t.Errorf("Expected 200 on valid telemetry POST, got %d", rrValid.Code)
	}

	if len(processor.events) != 1 {
		t.Fatalf("Expected 1 event passed to processor, got %d", len(processor.events))
	}
	if processor.events[0].identifier != "SENSOR-PRESSURE-01" {
		t.Errorf("Expected identifier SENSOR-PRESSURE-01, got %s", processor.events[0].identifier)
	}
	if processor.events[0].event.Message != "High pipeline pressure detected" {
		t.Errorf("Event message mismatch: %s", processor.events[0].event.Message)
	}

	// 4. Missing required field 'origin'
	missingOriginJSON := `{
		"level": "WARNING",
		"category": "HARDWARE",
		"message": "High pipeline pressure detected"
	}`
	reqMissingOrigin := httptest.NewRequest(http.MethodPost, "/api/telemetry", strings.NewReader(missingOriginJSON))
	rrMissingOrigin := httptest.NewRecorder()
	handler.HandleIngest(rrMissingOrigin, reqMissingOrigin)
	if rrMissingOrigin.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when 'origin' is missing, got %d", rrMissingOrigin.Code)
	}
}

func TestTelemetryHandler_HandlePair(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pair_test.db")
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()
	dRepo := storage.NewDeviceRepository(db)

	processor := &mockTelemetryProcessor{}
	handler := NewTelemetryHandler(processor, dRepo)

	// 1. Method Not Allowed (GET)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/pair", nil)
	rrGet := httptest.NewRecorder()
	handler.HandlePair(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 on GET /api/pair, got %d", rrGet.Code)
	}

	// 2. Missing Origin
	reqNoOrigin := httptest.NewRequest(http.MethodPost, "/api/pair", strings.NewReader(`{"name":"Sensor"}`))
	rrNoOrigin := httptest.NewRecorder()
	handler.HandlePair(rrNoOrigin, reqNoOrigin)
	if rrNoOrigin.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when origin missing, got %d", rrNoOrigin.Code)
	}

	// 3. Valid Pair Request
	reqValid := httptest.NewRequest(http.MethodPost, "/api/pair", strings.NewReader(`{"origin":"carina-edge","name":"Carina Sensor Node"}`))
	rrValid := httptest.NewRecorder()
	handler.HandlePair(rrValid, reqValid)

	if rrValid.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted from /api/pair, got %d: %s", rrValid.Code, rrValid.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rrValid.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse /api/pair response JSON: %v", err)
	}

	if _, exists := resp["api_key"]; exists {
		t.Fatal("SECURITY VIOLATION: API key must NOT be returned in /api/pair HTTP response!")
	}
	if resp["origin"] != "carina-edge" {
		t.Errorf("Expected origin carina-edge, got %v", resp["origin"])
	}

	// 4. Verify in DB
	dev, err := dRepo.GetByIdentifier("carina-edge")
	if err != nil || dev == nil {
		t.Fatalf("Device not found in db: %v", err)
	}
	if dev.Status != "PENDING" {
		t.Errorf("Expected status PENDING, got %s", dev.Status)
	}
	if !strings.HasPrefix(dev.ApiKey, "nox_live_") {
		t.Errorf("Expected api_key prefix nox_live_, got %s", dev.ApiKey)
	}
}

func TestTelemetryHandler_DeviceAuthenticationLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_lifecycle.db")
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()
	dRepo := storage.NewDeviceRepository(db)

	processor := &mockTelemetryProcessor{}
	handler := NewTelemetryHandler(processor, dRepo)

	// 1. Initial Greeting / Pair
	pairReq := httptest.NewRequest(http.MethodPost, "/api/pair", strings.NewReader(`{"origin":"synapse-01","name":"Synapse AI"}`))
	pairRec := httptest.NewRecorder()
	handler.HandlePair(pairRec, pairReq)
	if pairRec.Code != http.StatusAccepted {
		t.Fatalf("Expected 202, got %d", pairRec.Code)
	}

	dev, err := dRepo.GetByIdentifier("synapse-01")
	if err != nil || dev == nil {
		t.Fatalf("Device not found: %v", err)
	}
	apiKey := dev.ApiKey

	// 2. Ingest without token -> 401 Unauthorized
	payload := map[string]interface{}{
		"category":    "HARDWARE",
		"origin":      "synapse-01",
		"level":       "INFO",
		"message":     "System OK",
		"occurred_at": time.Now().Format(time.RFC3339),
	}
	pBytes, _ := json.Marshal(payload)

	reqNoToken := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(pBytes))
	recNoToken := httptest.NewRecorder()
	handler.HandleIngest(recNoToken, reqNoToken)
	if recNoToken.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized without token, got %d", recNoToken.Code)
	}

	// 3. Ingest with WRONG token -> 401 Unauthorized
	reqWrongToken := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(pBytes))
	reqWrongToken.Header.Set("X-Device-Token", "wrong-key")
	recWrongToken := httptest.NewRecorder()
	handler.HandleIngest(recWrongToken, reqWrongToken)
	if recWrongToken.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized with wrong token, got %d", recWrongToken.Code)
	}

	// 4. Ingest with CORRECT token -> 200 OK & Activation
	reqValid := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(pBytes))
	reqValid.Header.Set("X-Device-Token", apiKey)
	recValid := httptest.NewRecorder()
	handler.HandleIngest(recValid, reqValid)
	if recValid.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with correct token, got %d: %s", recValid.Code, recValid.Body.String())
	}

	// Verify device status is now ACTIVE
	devAfter, err := dRepo.GetByIdentifier("synapse-01")
	if err != nil || devAfter == nil {
		t.Fatalf("Device not found after activation: %v", err)
	}
	if devAfter.Status != "ACTIVE" {
		t.Errorf("Expected status ACTIVE, got %s", devAfter.Status)
	}

	// 5. Subsequent Ingestion with Bearer authorization header
	reqBearer := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(pBytes))
	reqBearer.Header.Set("Authorization", "Bearer "+apiKey)
	recBearer := httptest.NewRecorder()
	handler.HandleIngest(recBearer, reqBearer)
	if recBearer.Code != http.StatusOK {
		t.Errorf("Expected 200 OK with Bearer header, got %d", recBearer.Code)
	}

	// 6. Anti-Spoofing: X-Device-Origin header mismatch -> 403 Forbidden
	reqSpoof := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(pBytes))
	reqSpoof.Header.Set("X-Device-Token", apiKey)
	reqSpoof.Header.Set("X-Device-Origin", "synapse-fake")
	recSpoof := httptest.NewRecorder()
	handler.HandleIngest(recSpoof, reqSpoof)
	if recSpoof.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden on spoofed origin header, got %d", recSpoof.Code)
	}

	// 7. Anti-Replay: Timestamp older than 5 minutes -> 400 Bad Request
	oldPayload := map[string]interface{}{
		"category":    "HARDWARE",
		"origin":      "synapse-01",
		"level":       "INFO",
		"message":     "Old event",
		"occurred_at": time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
	}
	oldBytes, _ := json.Marshal(oldPayload)
	reqOld := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(oldBytes))
	reqOld.Header.Set("X-Device-Token", apiKey)
	recOld := httptest.NewRecorder()
	handler.HandleIngest(recOld, reqOld)
	if recOld.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request on replay timestamp (> 5 min drift), got %d", recOld.Code)
	}

	// 8. Unregistered device -> 403 Forbidden
	unregPayload := map[string]interface{}{
		"category": "HARDWARE",
		"origin":   "unknown-device",
		"level":    "INFO",
		"message":  "Hello",
	}
	unregBytes, _ := json.Marshal(unregPayload)
	reqUnreg := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewReader(unregBytes))
	recUnreg := httptest.NewRecorder()
	handler.HandleIngest(recUnreg, reqUnreg)
	if recUnreg.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for unknown device, got %d", recUnreg.Code)
	}
}

