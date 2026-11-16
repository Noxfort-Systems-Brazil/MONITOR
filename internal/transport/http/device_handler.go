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
// File: internal/transport/http/device_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/tunnel"
)

// DeviceHandler manages the CRUD operations for monitored systems.
type DeviceHandler struct {
	deviceRepo    domain.DeviceRepository
	tunnelService tunnel.Service
	auditRepo     domain.AuditRepository
}

// NewDeviceHandler initializes the handler with repository and tunnel service dependencies.
func NewDeviceHandler(dRepo domain.DeviceRepository, ts tunnel.Service) *DeviceHandler {
	return &DeviceHandler{
		deviceRepo:    dRepo,
		tunnelService: ts,
	}
}

// SetAuditRepository attaches an AuditRepository for recording device lifecycle mutations.
func (h *DeviceHandler) SetAuditRepository(repo domain.AuditRepository) {
	h.auditRepo = repo
}

func (h *DeviceHandler) logAudit(r *http.Request, action, details string) {
	logSecurityAuditEvent(h.auditRepo, r, action, details)
}

// ServePage renders the management list.
func (h *DeviceHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	devices, err := h.deviceRepo.GetAllDevices()
	if err != nil {
		log.Printf("[DEVICES] Error fetching devices: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "22100"
	}

	pushTarget := GetLocalIP()
	if h.tunnelService != nil {
		status := h.tunnelService.GetStatus()
		if status.TelemetryURL != "" {
			pushTarget = status.TelemetryURL
		}
	}

	var activeDevices []domain.Device
	var pendingDevices []domain.Device
	for _, d := range devices {
		if d.Status == "PENDING" {
			pendingDevices = append(pendingDevices, d)
		} else {
			// Redact active device tokens from rendered HTML; keys are shown only at creation or rotation.
			d.ApiKey = ""
			activeDevices = append(activeDevices, d)
		}
	}

	data := map[string]interface{}{
		"Title":          "System Management",
		"Devices":        activeDevices,
		"PendingDevices": pendingDevices,
		"ServerIP":       pushTarget,
		"ServerPort":     port,
	}

	if err := RenderPageWithRequest(w, r, "web/templates/devices.html", data); err != nil {
		log.Printf("[DEVICES] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}
}

func respondDeviceJSON(w http.ResponseWriter, status int, payload map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// HandleCreate provisions an ACTIVE device manually and returns a freshly generated one-time API key.
func (h *DeviceHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := strings.TrimSpace(r.FormValue("identifier"))
	name := strings.TrimSpace(r.FormValue("name"))
	if identifier == "" {
		respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "O identificador (origin) do dispositivo é obrigatório.",
		})
		return
	}
	if name == "" {
		name = identifier
	}

	apiKey, err := generateDeviceAPIKey()
	if err != nil {
		respondDeviceJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Falha ao gerar chave de segurança.",
		})
		return
	}

	dev, err := h.deviceRepo.CreateManualDevice(name, identifier, apiKey)
	if err != nil {
		respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	h.logAudit(r, "DEVICE_CREATED", fmt.Sprintf("Dispositivo cadastrado manualmente: %s (%s)", dev.Name, dev.Identifier))

	if wantsJSONResponse(r) {
		respondDeviceJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"identifier": dev.Identifier,
			"name":       dev.Name,
			"api_key":    apiKey,
			"message":    "Dispositivo cadastrado com sucesso! Copie a chave agora.",
		})
		return
	}
	Redirect(w, r, "/devices", http.StatusSeeOther)
}

// HandleUpdate renames a registered device (strictly requires POST).
func (h *DeviceHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := strings.TrimSpace(r.FormValue("identifier"))
	if identifier == "" {
		identifier = strings.TrimSpace(r.FormValue("id"))
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if identifier == "" || name == "" {
		if wantsJSONResponse(r) {
			respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Identificador e novo nome são obrigatórios.",
			})
			return
		}
		Redirect(w, r, "/devices", http.StatusSeeOther)
		return
	}

	if err := h.deviceRepo.UpdateDeviceName(identifier, name); err != nil {
		if wantsJSONResponse(r) {
			respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logAudit(r, "DEVICE_UPDATED", fmt.Sprintf("Dispositivo '%s' renomeado para '%s'", identifier, name))
	if wantsJSONResponse(r) {
		respondDeviceJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Nome do dispositivo atualizado com sucesso.",
		})
		return
	}
	Redirect(w, r, "/devices", http.StatusSeeOther)
}

// HandleToggle pauses or resumes monitoring (Maintenance Mode) for a device.
func (h *DeviceHandler) HandleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := strings.TrimSpace(r.FormValue("identifier"))
	if identifier == "" {
		identifier = strings.TrimSpace(r.FormValue("id"))
	}
	enabledStr := strings.TrimSpace(r.FormValue("enabled"))
	enabled := enabledStr == "true" || enabledStr == "1" || enabledStr == "on"

	if identifier == "" {
		if wantsJSONResponse(r) {
			respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Identificador do dispositivo é obrigatório.",
			})
			return
		}
		Redirect(w, r, "/devices", http.StatusSeeOther)
		return
	}

	if err := h.deviceRepo.SetDeviceEnabled(identifier, enabled); err != nil {
		if wantsJSONResponse(r) {
			respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action := "DEVICE_PAUSED"
	msg := "Monitoramento pausado (Modo Manutenção)."
	if enabled {
		action = "DEVICE_RESUMED"
		msg = "Monitoramento retomado com sucesso."
	}
	h.logAudit(r, action, fmt.Sprintf("Dispositivo '%s': enabled=%v", identifier, enabled))

	if wantsJSONResponse(r) {
		respondDeviceJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"enabled": enabled,
			"message": msg,
		})
		return
	}
	Redirect(w, r, "/devices", http.StatusSeeOther)
}

// HandleRotateKey generates a fresh API key for an existing device and returns it once for admin copy.
func (h *DeviceHandler) HandleRotateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := strings.TrimSpace(r.FormValue("identifier"))
	if identifier == "" {
		identifier = strings.TrimSpace(r.FormValue("id"))
	}
	if identifier == "" {
		respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Identificador do dispositivo é obrigatório.",
		})
		return
	}

	newKey, err := generateDeviceAPIKey()
	if err != nil {
		respondDeviceJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Falha ao gerar nova chave.",
		})
		return
	}

	if err := h.deviceRepo.RotateAPIKey(identifier, newKey); err != nil {
		respondDeviceJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	h.logAudit(r, "DEVICE_KEY_ROTATED", fmt.Sprintf("API Key rotacionada para '%s'", identifier))

	respondDeviceJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"identifier": identifier,
		"api_key":    newKey,
		"message":    "Chave rotacionada com sucesso! Atualize a configuração no dispositivo remoto.",
	})
}

// HandleDelete removes a system from monitoring (strictly requires POST).
func (h *DeviceHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := r.FormValue("id")
	if identifier == "" {
		identifier = r.URL.Query().Get("id")
	}

	if identifier != "" {
		log.Printf("[DEVICES] Requesting deletion for system: %s", identifier)
		if err := h.deviceRepo.DeleteDevice(identifier); err != nil {
			log.Printf("[DEVICES] Failed to delete %s: %v", identifier, err)
		} else {
			log.Printf("[DEVICES] System %s removed successfully.", identifier)
			h.logAudit(r, "DEVICE_DELETE", fmt.Sprintf("Dispositivo removido: '%s'", identifier))
		}
	} else {
		log.Println("[DEVICES] Delete request missing 'id' parameter.")
	}

	// Refresh the list
	Redirect(w, r, "/devices", http.StatusSeeOther)
}
