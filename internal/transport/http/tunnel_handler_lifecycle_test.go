// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/tunnel_handler_lifecycle_test.go
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: Lifecycle tests (start, stop, disconnect) for TunnelHandler.

package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"noxfort-monitor-server/internal/domain"
)

func TestTunnelHandler_HandleStartWithoutToken(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 when starting without token, got %d", rec.Code)
	}
}

func TestTunnelHandler_HandleStartFailure(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:  "token",
			DuckDNSDomain: "domain",
		},
	}
	mockSvc := &mockTunnelService{
		startErr: errors.New("simulated error"),
	}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStart(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 when start fails, got %d", rec.Code)
	}
}

func TestTunnelHandler_HandleStartSuccess(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:   "valid-token",
			DuckDNSDomain:  "valid-domain",
			DuckDNSEnabled: false,
		},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/start", nil)
	rec := httptest.NewRecorder()

	handler.HandleStart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if !saved.DuckDNSEnabled {
		t.Errorf("Expected DuckDNSEnabled to be true after start, got false")
	}
}

func TestTunnelHandler_HandleStop(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:   "tok",
			DuckDNSDomain:  "dom",
			DuckDNSEnabled: true,
		},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/stop", nil)
	rec := httptest.NewRecorder()

	handler.HandleStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if saved.DuckDNSEnabled {
		t.Errorf("Expected DuckDNSEnabled to be false after stop, got true")
	}
}

func TestTunnelHandler_HandleDisconnect(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:   "secret-tok",
			DuckDNSDomain:  "test-sub",
			DuckDNSEnabled: true,
		},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/disconnect", nil)
	rec := httptest.NewRecorder()

	handler.HandleDisconnect(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if saved.DuckDNSToken != "" {
		t.Errorf("Expected DuckDNSToken to be cleared, got '%s'", saved.DuckDNSToken)
	}
	if saved.DuckDNSDomain != "" {
		t.Errorf("Expected DuckDNSDomain to be cleared, got '%s'", saved.DuckDNSDomain)
	}
	if saved.DuckDNSEnabled {
		t.Errorf("Expected DuckDNSEnabled to be false after disconnect, got true")
	}
}
