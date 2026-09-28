// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
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
// File: internal/transport/http/tunnel_handler_test.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-04 (SOLID: MockTunnelService isolation)

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/tunnel"
)

func TestTunnelHandler_HandleStatus(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			NgrokAuthToken: "tok_123",
			NgrokDomain:    "test.ngrok-free.app",
		},
	}
	mockSvc := &mockTunnelService{
		status: tunnel.Status{State: tunnel.StateOffline},
	}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
	rec := httptest.NewRecorder()

	handler.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	var status tunnel.Status
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if status.State != tunnel.StateOffline {
		t.Errorf("Expected initial state OFFLINE, got %s", status.State)
	}
}

func TestTunnelHandler_HandleSave(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	formData := url.Values{}
	formData.Set("duckdns_token", "my_secret_token_123")
	formData.Set("duckdns_domain", "noxfort-lab")
	formData.Set("duckdns_enabled", "on")

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.HandleSave(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("Expected redirect status 303, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if saved.DuckDNSToken != "my_secret_token_123" {
		t.Errorf("Expected saved token 'my_secret_token_123', got '%s'", saved.DuckDNSToken)
	}
	if saved.DuckDNSDomain != "noxfort-lab" {
		t.Errorf("Expected saved domain 'noxfort-lab', got '%s'", saved.DuckDNSDomain)
	}
	if !saved.DuckDNSEnabled {
		t.Errorf("Expected DuckDNSEnabled to be true, got false")
	}

	if mockSvc.startedToken != "my_secret_token_123" {
		t.Errorf("Expected mock service to have started with token 'my_secret_token_123', got '%s'", mockSvc.startedToken)
	}
}

func TestTunnelHandler_HandleSave_DoesNotAutoConnect(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	formData := url.Values{}
	formData.Set("duckdns_token", "my_saved_token")
	formData.Set("duckdns_domain", "noxfort-saved")

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleSave(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	saved, _ := repo.GetSettings()
	if saved.DuckDNSToken != "my_saved_token" || saved.DuckDNSDomain != "noxfort-saved" {
		t.Errorf("Expected token and domain saved, got token=%s, domain=%s", saved.DuckDNSToken, saved.DuckDNSDomain)
	}
	if saved.DuckDNSEnabled {
		t.Errorf("Expected DuckDNSEnabled to remain false when saving without auto-start")
	}
	if mockSvc.startedToken != "" {
		t.Errorf("Expected tunnelService NOT to start, but was started with %s", mockSvc.startedToken)
	}
}

func TestTunnelHandler_HandleSave_Multipart(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("duckdns_token", "multipart_token")
	_ = w.WriteField("duckdns_domain", "multipart_subdomain")
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/save", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleSave(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	saved, _ := repo.GetSettings()
	if saved.DuckDNSToken != "multipart_token" {
		t.Errorf("Expected saved token 'multipart_token', got '%s'", saved.DuckDNSToken)
	}
	if saved.DuckDNSDomain != "multipart_subdomain" {
		t.Errorf("Expected saved domain 'multipart_subdomain', got '%s'", saved.DuckDNSDomain)
	}
}

func TestTunnelHandler_HandleTest_MissingCredentials(t *testing.T) {
	repo := &mockSettingsRepoForHandler{settings: &domain.Settings{}}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/test", nil)
	rec := httptest.NewRecorder()

	handler.HandleTest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 when testing without credentials, got %d", rec.Code)
	}
}

func TestTunnelHandler_HandleTest_Success(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:  "tok-123",
			DuckDNSDomain: "my-sub",
		},
	}
	mockSvc := &mockTunnelService{}
	handler := NewTunnelHandler(repo, mockSvc)

	formData := url.Values{}
	formData.Set("duckdns_token", "tok-123")
	formData.Set("duckdns_domain", "my-sub")

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/test", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.HandleTest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var result tunnel.TestResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON result: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected result.Success to be true")
	}
	if mockSvc.testedToken != "tok-123" || mockSvc.testedDom != "my-sub" {
		t.Errorf("Expected service to be called with tok-123 and my-sub, got %s and %s", mockSvc.testedToken, mockSvc.testedDom)
	}
}

func TestTunnelHandler_HandleTest_ServiceError(t *testing.T) {
	repo := &mockSettingsRepoForHandler{
		settings: &domain.Settings{
			DuckDNSToken:  "tok-123",
			DuckDNSDomain: "my-sub",
		},
	}
	mockSvc := &mockTunnelService{
		testErr: errors.New("duckdns rejection: KO"),
	}
	handler := NewTunnelHandler(repo, mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/tunnel/test", nil)
	rec := httptest.NewRecorder()

	handler.HandleTest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 when test fails, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "duckdns rejection: KO") {
		t.Errorf("Expected body to contain error message, got %s", rec.Body.String())
	}
}
