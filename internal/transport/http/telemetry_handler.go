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
// File: internal/transport/http/telemetry_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"
)

// PairRequest represents the initial greeting ("Oi") from an agent.
type PairRequest struct {
	Origin string `json:"origin"`
	Name   string `json:"name"`
}

// TelemetryHandler processes HTTP REST telemetry ingest and device pairing payloads.
type TelemetryHandler struct {
	processor  monitor.EventProcessor
	deviceRepo domain.DeviceRepository
}

// NewTelemetryHandler creates a new TelemetryHandler instance.
func NewTelemetryHandler(processor monitor.EventProcessor, deviceRepo domain.DeviceRepository) *TelemetryHandler {
	return &TelemetryHandler{
		processor:  processor,
		deviceRepo: deviceRepo,
	}
}

// HandlePair registers an agent pairing request with a 5-minute TTL.
// Crucially, the generated API key is NOT returned in the HTTP response (Out-of-Band delivery).
func (h *TelemetryHandler) HandlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Payload size limit (Anti-DoS)
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	var req PairRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON schema: %v", err), http.StatusBadRequest)
		return
	}

	req.Origin = strings.TrimSpace(req.Origin)
	if req.Origin == "" {
		http.Error(w, "Field 'origin' is mandatory", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = req.Origin
	}

	// Anti-DoS/Spam check on pending pairings
	if h.deviceRepo != nil {
		devices, err := h.deviceRepo.GetAllDevices()
		if err == nil {
			pendingCount := 0
			for _, d := range devices {
				if d.Status == "PENDING" {
					pendingCount++
				}
			}
			if pendingCount >= 50 {
				http.Error(w, "Too many pending pairing requests. Please approve or cancel existing sessions.", http.StatusTooManyRequests)
				return
			}
		}
	}

	// 2. Generate cryptographically secure API Key (32 random bytes = 64 hex chars)
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		http.Error(w, "Internal error generating security token", http.StatusInternalServerError)
		return
	}
	apiKey := "nox_live_" + hex.EncodeToString(rawBytes)
	expiresAt := time.Now().Add(5 * time.Minute)

	// 3. Save as PENDING in the active database
	device := &domain.Device{
		Identifier: req.Origin,
		Name:       req.Name,
		ApiKey:     apiKey,
		Status:     "PENDING",
		ExpiresAt:  &expiresAt,
		Enabled:    true,
	}

	if h.deviceRepo != nil {
		if err := h.deviceRepo.CreateOrUpdatePending(device); err != nil {
			log.Printf("[SECURITY] Error saving pending device %s: %v", req.Origin, err)
			http.Error(w, "Failed to persist pairing session", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("[SECURITY] Device '%s' requested pairing. API Key generated on console (TTL: 5m).", req.Origin)

	// 4. Return 202 Accepted WITHOUT the API key
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "pending_admin_approval",
		"message":            "Pairing registered. Copy the API key generated on the Monitor console to your device settings within 5 minutes.",
		"origin":             req.Origin,
		"expires_in_seconds": 300,
	})
}

// HandleIngest receives telemetry event data, verifies tokens, and activates pending devices.
func (h *TelemetryHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Limite de tamanho de payload (64 KB)
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	// 2. Extração do Token de Autenticação (Header X-Device-Token ou Authorization Bearer)
	token := r.Header.Get("X-Device-Token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// 3. Decodificador Estrito de JSON (Rejeita campos desconhecidos)
	var event domain.IncomingEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON schema: %v", err), http.StatusBadRequest)
		return
	}

	// 4. Validação Semântica de Campos Obrigatórios
	event.Origin = strings.TrimSpace(event.Origin)
	if event.Origin == "" {
		http.Error(w, "Missing 'origin' identifier", http.StatusBadRequest)
		return
	}

	if event.Category != domain.CategoryHardware && event.Category != domain.CategorySoftware {
		http.Error(w, "Invalid 'category'. Must be 'HARDWARE' or 'SOFTWARE'", http.StatusUnprocessableEntity)
		return
	}

	if event.Level != domain.LevelInfo && event.Level != domain.LevelWarning && event.Level != domain.LevelCritical {
		http.Error(w, "Invalid 'level'. Must be 'INFO', 'WARNING' or 'CRITICAL'", http.StatusUnprocessableEntity)
		return
	}

	// 5. Anti-Spoofing de Cabeçalho
	headerOrigin := strings.TrimSpace(r.Header.Get("X-Device-Origin"))
	if headerOrigin != "" && headerOrigin != event.Origin {
		http.Error(w, "Forbidden: Header X-Device-Origin mismatch with payload origin", http.StatusForbidden)
		return
	}

	// 6. Janela de Validação Temporal (Anti-Replay Attack: limite de ±5 minutos)
	if !event.OccurredAt.IsZero() {
		drift := math.Abs(time.Since(event.OccurredAt).Seconds())
		if drift > 300 {
			http.Error(w, "Bad Request: Timestamp expired or out of allowed window (±5 min)", http.StatusBadRequest)
			return
		}
	}

	// 7. Autenticação e Ativação do Dispositivo no Banco de Dados
	if h.deviceRepo != nil {
		device, err := h.deviceRepo.GetByIdentifier(event.Origin)
		if err != nil {
			log.Printf("[SECURITY] Error querying device %s: %v", event.Origin, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if device == nil {
			http.Error(w, "Device not recognized. Please pair first via POST /api/pair", http.StatusForbidden)
			return
		}

		// Cenário A: Dispositivo ainda está em estado PENDING (Primeiro Envio com a Chave)
		if device.Status == "PENDING" {
			if token == "" {
				http.Error(w, "Unauthorized: Device is pending pairing. Missing X-Device-Token", http.StatusUnauthorized)
				return
			}

			activated, err := h.deviceRepo.ActivateDevice(event.Origin, token)
			if err != nil {
				http.Error(w, fmt.Sprintf("Forbidden: %v", err), http.StatusForbidden)
				return
			}
			if !activated {
				http.Error(w, "Unauthorized: Invalid pairing API key", http.StatusUnauthorized)
				return
			}

			log.Printf("[SECURITY] ✅ Device '%s' successfully paired and activated on first ingestion!", event.Origin)
		} else if device.Status == "ACTIVE" {
			// Cenário B: Dispositivo já é ACTIVE
			if device.ApiKey != "" {
				if token == "" || token != device.ApiKey {
					http.Error(w, "Unauthorized: Invalid or missing X-Device-Token", http.StatusUnauthorized)
					return
				}
			}

			if !device.Enabled {
				http.Error(w, "Device disabled by administrator (Maintenance Mode)", http.StatusForbidden)
				return
			}
		}
	}

	// 8. Encaminhar para o EventProcessor
	if h.processor != nil {
		h.processor.ProcessEvent(event.Origin, &event)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

