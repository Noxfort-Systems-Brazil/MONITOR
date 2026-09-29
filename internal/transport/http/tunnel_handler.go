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
// File: internal/transport/http/tunnel_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-14 (SOLID Refactor: Interface Segregation, DRY & Lean Transport)

package http

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/tunnel"
)

// SettingsStore defines the segregated persistence contract needed by TunnelHandler (ISP).
type SettingsStore interface {
	GetSettings() (*domain.Settings, error)
	SaveSettings(s *domain.Settings) error
}

// TunnelHandler manages the Remote Access (DuckDNS Tunnel) page and API endpoints.
type TunnelHandler struct {
	settingsRepo  SettingsStore
	tunnelService tunnel.Service
}

// NewTunnelHandler creates a new TunnelHandler instance with injected dependencies (DIP).
func NewTunnelHandler(settingsRepo SettingsStore, ts tunnel.Service) *TunnelHandler {
	return &TunnelHandler{settingsRepo: settingsRepo, tunnelService: ts}
}

// ServePage renders the dedicated Remote Access HTML page.
func (h *TunnelHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[REMOTE] Error loading settings: %v", err)
		http.Error(w, "Failed to load system settings", http.StatusInternalServerError)
		return
	}

	status := h.tunnelService.GetStatus()
	localIP, localPort := GetLocalIP(), status.LocalPort
	if localPort == "" {
		if localPort = os.Getenv("PORT"); localPort == "" {
			localPort = "22100"
		}
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/remote.html"),
	)
	if err != nil {
		log.Printf("[REMOTE] Template rendering error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, map[string]interface{}{
		"Title":       "Remote Access",
		"Settings":    settings,
		"Status":      status,
		"LocalIP":     localIP,
		"LocalPort":   localPort,
		"LocalURL":    "http://" + localIP + ":" + localPort + "/api/telemetry",
		"BinaryFound": h.tunnelService.IsBinaryAvailable(),
	})
}

// HandleStatus returns the current live status of the tunnel as JSON.
func (h *TunnelHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, h.tunnelService.GetStatus())
}

// HandleSave updates the tunnel credentials in the database without auto-connecting unless explicitly requested.
func (h *TunnelHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if !h.isPost(w, r) {
		return
	}
	_ = r.ParseMultipartForm(32 << 20)
	if err := r.ParseForm(); err != nil && r.MultipartForm == nil {
		http.Error(w, "Invalid Form Data", http.StatusBadRequest)
		return
	}

	token := tunnel.CleanDuckDNSToken(r.FormValue("duckdns_token"))
	domainName := tunnel.CleanDuckDNSDomain(r.FormValue("duckdns_domain"))
	autoStart := r.FormValue("duckdns_enabled") == "on" || r.FormValue("duckdns_enabled") == "true" || r.FormValue("auto_start") == "true"

	err := h.mutateSettings(func(s *domain.Settings) {
		s.DuckDNSToken, s.DuckDNSDomain = token, domainName
		if autoStart {
			s.DuckDNSEnabled = true
		} else if token == "" || domainName == "" {
			s.DuckDNSEnabled = false
		}
		s.NgrokAuthToken, s.NgrokDomain, s.NgrokEnabled = "", "", false
	})
	if err != nil {
		http.Error(w, "Failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if autoStart && token != "" {
		_ = h.tunnelService.Start(token, domainName)
	} else if token == "" {
		_ = h.tunnelService.Stop()
	}

	if strings.Contains(r.Header.Get("Accept"), "json") || r.Header.Get("X-Requested-With") != "" {
		h.respondResult(w, http.StatusOK, true, "Configurações salvas com sucesso.")
		return
	}
	http.Redirect(w, r, "/remote?success=1", http.StatusSeeOther)
}

// HandleStart triggers tunnel startup on demand and marks it enabled in settings.
func (h *TunnelHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if !h.isPost(w, r) {
		return
	}
	_ = r.ParseMultipartForm(32 << 20)
	_ = r.ParseForm()

	token := tunnel.CleanDuckDNSToken(r.FormValue("duckdns_token"))
	domainName := tunnel.CleanDuckDNSDomain(r.FormValue("duckdns_domain"))

	if token == "" || domainName == "" {
		if settings, _ := h.settingsRepo.GetSettings(); settings != nil {
			if token == "" {
				token = settings.DuckDNSToken
			}
			if domainName == "" {
				domainName = settings.DuckDNSDomain
			}
		}
	}

	if token == "" || domainName == "" {
		h.respondResult(w, http.StatusBadRequest, false, "Token não configurado. Por favor, configure seu token DuckDNS primeiro.")
		return
	}

	// Always persist credentials immediately so they are never lost even if start fails
	_ = h.mutateSettings(func(s *domain.Settings) {
		s.DuckDNSToken = token
		s.DuckDNSDomain = domainName
	})

	if err := h.tunnelService.Start(token, domainName); err != nil {
		h.respondResult(w, http.StatusInternalServerError, false, err.Error())
		return
	}

	_ = h.mutateSettings(func(s *domain.Settings) {
		s.DuckDNSToken = token
		s.DuckDNSDomain = domainName
		s.DuckDNSEnabled = true
	})
	h.respondResult(w, http.StatusOK, true, "Serviço DuckDNS iniciado com sucesso.")
}

// HandleStop stops the tunnel on demand and disables auto-start on boot.
func (h *TunnelHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	if !h.isPost(w, r) {
		return
	}
	_ = h.tunnelService.Stop()
	_ = h.mutateSettings(func(s *domain.Settings) { s.DuckDNSEnabled = false })
	h.respondResult(w, http.StatusOK, true, "Serviço DuckDNS desconectado/pausado com sucesso.")
}

// HandleDisconnect completely unlinks DuckDNS credentials and terminates the connection.
func (h *TunnelHandler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	if !h.isPost(w, r) {
		return
	}
	_ = h.tunnelService.Stop()
	if err := h.mutateSettings(func(s *domain.Settings) {
		s.DuckDNSToken, s.DuckDNSDomain, s.DuckDNSEnabled = "", "", false
	}); err != nil {
		h.respondResult(w, http.StatusInternalServerError, false, "Falha ao remover credenciais: "+err.Error())
		return
	}
	h.respondResult(w, http.StatusOK, true, "Credenciais do DuckDNS removidas e serviço desconectado com sucesso.")
}

// HandleTest verifies tunnel credentials and connectivity via the injected service.
func (h *TunnelHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if !h.isPost(w, r) {
		return
	}
	_ = r.ParseMultipartForm(32 << 20)
	_ = r.ParseForm()
	token := tunnel.CleanDuckDNSToken(r.FormValue("duckdns_token"))
	domainName := tunnel.CleanDuckDNSDomain(r.FormValue("duckdns_domain"))
	if (token == "" || domainName == "") && h.settingsRepo != nil {
		if s, _ := h.settingsRepo.GetSettings(); s != nil {
			if token == "" {
				token = s.DuckDNSToken
			}
			if domainName == "" {
				domainName = s.DuckDNSDomain
			}
		}
	}

	if token == "" || domainName == "" {
		h.respondResult(w, http.StatusBadRequest, false, "Token e Subdomínio são obrigatórios para testar a conexão.")
		return
	}

	result, err := h.tunnelService.TestConnection(r.Context(), token, domainName)
	if err != nil {
		h.respondResult(w, http.StatusBadRequest, false, err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// ─── Helpers (DRY) ───

func (h *TunnelHandler) mutateSettings(fn func(*domain.Settings)) error {
	settings, err := h.settingsRepo.GetSettings()
	if err != nil || settings == nil {
		return err
	}
	fn(settings)
	return h.settingsRepo.SaveSettings(settings)
}

func (h *TunnelHandler) isPost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func (h *TunnelHandler) respondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *TunnelHandler) respondResult(w http.ResponseWriter, code int, ok bool, msg string) {
	payload := map[string]interface{}{"success": ok}
	if ok {
		payload["message"] = msg
	} else {
		payload["error"] = msg
	}
	h.respondJSON(w, code, payload)
}
