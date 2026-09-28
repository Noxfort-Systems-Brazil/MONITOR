// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/routes.go
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: HTTP route declarations and multiplexer setup for the desktop application.

package http

import (
	"encoding/json"
	"net/http"

	"noxfort-monitor-server/internal/appdir"
)

// Handler configures the HTTP ServeMux and applies the security middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 1. Static Assets (Public)
	fs := http.FileServer(http.Dir(appdir.Path("web/static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 1b. Observability & Health Checks (Public / Monitoring Probes)
	if s.healthHandler != nil {
		mux.HandleFunc("/healthz", s.healthHandler.ServeLiveness)
		mux.HandleFunc("/api/health", s.healthHandler.ServeHealth)
	}
	if s.metricsHandler != nil {
		mux.HandleFunc("/metrics", s.metricsHandler.ServeMetrics)
	}

	// 2. Authentication Pages & APIs
	mux.HandleFunc("/login", s.authHandler.ServeLogin)
	mux.HandleFunc("/register", s.authHandler.ServeRegister)
	mux.HandleFunc("/api/auth/login", s.authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/register", s.userHandler.HandleRegister)
	mux.HandleFunc("/api/auth/logout", s.authHandler.HandleLogout)
	mux.HandleFunc("/api/auth/status", s.authHandler.HandleStatus)

	// 3. IoT Telemetry API (HTTP POST Ingest)
	mux.HandleFunc("/api/telemetry", s.telemetryHandler.HandleIngest)

	// 4. Protected Application Routes
	mux.HandleFunc("/", s.dashboardHandler.ServePage)

	// System Management
	mux.HandleFunc("/devices", s.deviceHandler.ServePage)
	mux.HandleFunc("/devices/delete", s.deviceHandler.HandleDelete)

	// Response Team (Contacts)
	mux.HandleFunc("/contacts", s.contactHandler.ServePage)
	mux.HandleFunc("/contacts/create", s.contactHandler.HandleCreate)
	mux.HandleFunc("/contacts/update", s.contactHandler.HandleUpdate)
	mux.HandleFunc("/contacts/delete", s.contactHandler.HandleDelete)

	// Remote Access & Ingestion Tunnel (DuckDNS)
	mux.HandleFunc("/remote", s.tunnelHandler.ServePage)
	mux.HandleFunc("/api/tunnel/status", s.tunnelHandler.HandleStatus)
	mux.HandleFunc("/api/tunnel/save", s.tunnelHandler.HandleSave)
	mux.HandleFunc("/api/tunnel/start", s.tunnelHandler.HandleStart)
	mux.HandleFunc("/api/tunnel/stop", s.tunnelHandler.HandleStop)
	mux.HandleFunc("/api/tunnel/disconnect", s.tunnelHandler.HandleDisconnect)
	mux.HandleFunc("/api/tunnel/test", s.tunnelHandler.HandleTest)

	// Settings
	mux.HandleFunc("/settings", s.settingsHandler.ServePage)
	mux.HandleFunc("/settings/save", s.settingsHandler.HandleSave)
	mux.HandleFunc("/settings/test", s.settingsHandler.HandleTest)
	mux.HandleFunc("/settings/test-telegram", s.settingsHandler.HandleTestTelegram)

	// Database Management & Server Configuration
	if s.databaseHandler != nil {
		mux.HandleFunc("/server", s.databaseHandler.ServePage)
		mux.HandleFunc("/api/settings/database/status", s.databaseHandler.HandleStatus)
		mux.HandleFunc("/api/settings/database/test", s.databaseHandler.HandleTest)
		mux.HandleFunc("/api/settings/database/save", s.databaseHandler.HandleSave)
		mux.HandleFunc("/api/settings/database/provision-user", s.databaseHandler.HandleProvisionUser)
		mux.HandleFunc("/api/settings/database/backup", s.databaseHandler.HandleBackup)
		mux.HandleFunc("/api/settings/database/backups", s.databaseHandler.HandleListBackups)
	}

	// Audit Trail
	if s.auditHandler != nil {
		mux.HandleFunc("/audit", s.auditHandler.ServePage)
		mux.HandleFunc("/api/audit/security", s.auditHandler.HandleSecurityLogs)
		mux.HandleFunc("/api/audit/alerts", s.auditHandler.HandleAlertLogs)
		mux.HandleFunc("/api/audit/transitions", s.auditHandler.HandleTransitionLogs)
	}

	// Account Management
	mux.HandleFunc("/users", s.userHandler.ServePage)
	mux.HandleFunc("/api/users", s.userHandler.HandleList)
	mux.HandleFunc("/api/users/create", s.userHandler.HandleCreateUser)
	mux.HandleFunc("/api/users/delete", s.userHandler.HandleDelete)

	// 5. Open External Links in Default Browser
	mux.HandleFunc("/api/open-external", s.browserHandler.HandleOpenExternal)

	// 6. Window Controls (Fallback for standalone browser/headless mode)
	mux.HandleFunc("/api/window/toggle-fullscreen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"desktop": false,
		})
	})
	mux.HandleFunc("/api/window/exit-fullscreen", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"desktop": false,
		})
	})

	// Wrap routing tree with Auth and RBAC middleware
	return s.authMiddleware.Wrap(mux)
}

// DesktopHandler returns the full graphical user interface routing tree,
// utilized exclusively by the native desktop WebKit container (Wails).
func (s *Server) DesktopHandler() http.Handler {
	return s.Handler()
}
