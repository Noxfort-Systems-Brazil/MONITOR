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
// File: internal/transport/http/server.go
// Author: Gabriel Moraes
// Date: 2026-01-19
// Modified: 2026-09-04 (SOLID Refactor)

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/tunnel"
)

// SecurityService abstracts authentication, user management, and session validation (ISP / DIP).
type SecurityService interface {
	AuthService
	UserManagementService
	SessionValidator
}

// Server is the HTTP router and service lifecycle orchestrator.
type Server struct {
	addr string

	// Modular Handlers
	dashboardHandler *DashboardHandler
	deviceHandler    *DeviceHandler
	contactHandler   *ContactHandler
	settingsHandler  *SettingsHandler
	databaseHandler  *DatabaseHandler
	auditHandler     *AuditHandler
	authHandler      *AuthHandler
	userHandler      *UserHandler
	telemetryHandler *TelemetryHandler
	browserHandler   *BrowserHandler
	tunnelHandler    *TunnelHandler
	healthHandler    *HealthHandler
	metricsHandler   *MetricsHandler

	authMiddleware *AuthMiddleware

	httpServer *http.Server
}

// NewServer initializes the HTTP server, assembling modular sub-handlers and middleware.
// Strictly adheres to Dependency Inversion Principle (DIP) by depending solely on interfaces.
func NewServer(
	addr string,
	dRepo domain.DeviceRepository,
	tRepo domain.TelemetryRepository,
	cRepo domain.ContactRepository,
	sRepo domain.SettingsRepository,
	sm monitor.EventProcessor,
	tester ConnectionTester,
	secService SecurityService,
	tm tunnel.Service,
	dbService DatabaseService,
	auditRepo domain.AuditRepository,
) *Server {
	var authHandler *AuthHandler
	var userHandler *UserHandler
	if secService != nil {
		authHandler = NewAuthHandler(secService)
		userHandler = NewUserHandler(secService, secService)
	}

	var dbH *DatabaseHandler
	if dbService != nil {
		dbH = NewDatabaseHandler(dbService, auditRepo)
	}

	var auditH *AuditHandler
	if auditRepo != nil {
		auditH = NewAuditHandler(auditRepo)
	}

	healthH := NewHealthHandler(dbService, nil, dRepo)
	metricsH := NewMetricsHandler(dbService, nil, dRepo)

	return &Server{
		addr:             addr,
		dashboardHandler: NewDashboardHandler(dRepo, tRepo),
		deviceHandler:    NewDeviceHandler(dRepo, tm),
		contactHandler:   NewContactHandler(cRepo),
		settingsHandler:  NewSettingsHandler(sRepo, tester),
		databaseHandler:  dbH,
		auditHandler:     auditH,
		authHandler:      authHandler,
		userHandler:      userHandler,
		telemetryHandler: NewTelemetryHandler(sm),
		browserHandler:   NewBrowserHandler(),
		tunnelHandler:    NewTunnelHandler(sRepo, tm),
		healthHandler:    healthH,
		metricsHandler:   metricsH,
		authMiddleware:   NewAuthMiddleware(authHandler),
	}
}

// SetMQTTStatus attaches an MQTT client status provider for health probes and Prometheus metrics.
func (s *Server) SetMQTTStatus(mqttStatus MQTTStatusProvider) {
	if s.healthHandler != nil {
		s.healthHandler.mqttStatus = mqttStatus
	}
	if s.metricsHandler != nil {
		s.metricsHandler.mqttStatus = mqttStatus
	}
}

// SetBackupService attaches a backup service to the database handler.
func (s *Server) SetBackupService(bs BackupService) {
	if s.databaseHandler != nil {
		s.databaseHandler.SetBackupService(bs)
	}
}

// ExternalIngestionHandler returns an HTTP handler restricted strictly to IoT telemetry ingestion.
// Direct browser access to dashboard/admin UI routes over external HTTP ports is completely blocked,
// enforcing that user interaction happens exclusively inside the native desktop application.
func (s *Server) ExternalIngestionHandler() http.Handler {
	mux := http.NewServeMux()

	// 1. Telemetry Ingestion for field agents (Carina, Synapse, IoT)
	mux.HandleFunc("/api/telemetry", s.telemetryHandler.HandleIngest)

	// 2. All other routes: Block browser access with 403 Forbidden
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Forbidden",
				"message": "O Noxfort Monitor opera exclusivamente como aplicativo desktop nativo. O acesso à interface pelo navegador está desativado.",
			})
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Noxfort Monitor™ · Acesso Restrito</title>
    <style>
        body {
            background-color: #0d1117;
            color: #c9d1d9;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            padding: 20px;
            box-sizing: border-box;
        }
        .container {
            max-width: 520px;
            width: 100%;
            padding: 36px 30px;
            background-color: #161b22;
            border: 1px solid #30363d;
            border-radius: 12px;
            text-align: center;
            box-shadow: 0 16px 36px rgba(0,0,0,0.6);
        }
        h2 { color: #f0ad4e; margin: 0 0 16px 0; font-size: 1.5rem; }
        p { color: #8b949e; line-height: 1.5; font-size: 0.95rem; margin: 0 0 12px 0; }
        .badge {
            display: inline-block;
            background: #21262d;
            color: #58a6ff;
            border: 1px solid #30363d;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 0.85rem;
            margin-top: 16px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>Noxfort Monitor™</h2>
        <p><strong>Acesso via navegador desativado.</strong></p>
        <p>O Noxfort Monitor opera exclusivamente como um <strong>aplicativo desktop nativo</strong>.</p>
        <p>Para interagir com o painel de controle, utilize a janela do aplicativo aberta no computador.</p>
        <div class="badge">Porta de Telemetria Ativa: POST /api/telemetry</div>
    </div>
</body>
</html>`)
	})

	return mux
}

// Run configures the external telemetry ingestion listener and starts the listening loop.
func (s *Server) Run() error {
	handler := s.ExternalIngestionHandler()

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("🌍 External Telemetry Listener running on %s (Ingestion only: POST /api/telemetry)", s.addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
