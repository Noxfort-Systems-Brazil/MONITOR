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
// File: internal/transport/http/device_handler_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

type mockDeviceRepoForHandler struct {
	devices []domain.Device
	deleted string
}

func (m *mockDeviceRepoForHandler) GetAllDevices() ([]domain.Device, error) {
	return m.devices, nil
}

func (m *mockDeviceRepoForHandler) UpdateLastSeen(identifier string, lastSeen time.Time) error {
	return nil
}

func (m *mockDeviceRepoForHandler) DeleteDevice(identifier string) error {
	m.deleted = identifier
	return nil
}

func (m *mockDeviceRepoForHandler) GetByIdentifier(identifier string) (*domain.Device, error) {
	for _, d := range m.devices {
		if d.Identifier == identifier {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *mockDeviceRepoForHandler) CreateOrUpdatePending(device *domain.Device) error {
	return nil
}

func (m *mockDeviceRepoForHandler) ActivateDevice(identifier string, token string) (bool, error) {
	return true, nil
}

func (m *mockDeviceRepoForHandler) ValidateToken(identifier string, token string) (bool, error) {
	return true, nil
}

func (m *mockDeviceRepoForHandler) CleanupExpiredPending() error {
	return nil
}

func (m *mockDeviceRepoForHandler) CreateManualDevice(name, identifier, apiKey string) (*domain.Device, error) {
	d := domain.Device{
		ID:         int64(len(m.devices) + 1),
		Name:       name,
		Identifier: identifier,
		ApiKey:     apiKey,
		Status:     "ACTIVE",
		Enabled:    true,
	}
	m.devices = append(m.devices, d)
	return &d, nil
}

func (m *mockDeviceRepoForHandler) UpdateDeviceName(identifier, name string) error {
	for i := range m.devices {
		if m.devices[i].Identifier == identifier {
			m.devices[i].Name = name
			return nil
		}
	}
	return nil
}

func (m *mockDeviceRepoForHandler) SetDeviceEnabled(identifier string, enabled bool) error {
	for i := range m.devices {
		if m.devices[i].Identifier == identifier {
			m.devices[i].Enabled = enabled
			return nil
		}
	}
	return nil
}

func (m *mockDeviceRepoForHandler) RotateAPIKey(identifier, newApiKey string) error {
	for i := range m.devices {
		if m.devices[i].Identifier == identifier {
			m.devices[i].ApiKey = newApiKey
			m.devices[i].Status = "ACTIVE"
			return nil
		}
	}
	return nil
}

func TestDeviceHandler_HandleDeleteRequiresPOST(t *testing.T) {
	mockRepo := &mockDeviceRepoForHandler{}
	handler := NewDeviceHandler(mockRepo, nil)

	// 1. GET request must return 405 Method Not Allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/devices/delete?id=synapse", nil)
	recGet := httptest.NewRecorder()
	handler.HandleDelete(recGet, reqGet)

	if recGet.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405 Method Not Allowed on GET, got %d", recGet.Code)
	}
	if mockRepo.deleted != "" {
		t.Errorf("Expected no device to be deleted on GET request")
	}

	// 2. POST request must delete device and redirect (303)
	form := url.Values{}
	form.Set("id", "synapse")
	reqPost := httptest.NewRequest(http.MethodPost, "/devices/delete", strings.NewReader(form.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recPost := httptest.NewRecorder()

	handler.HandleDelete(recPost, reqPost)

	if recPost.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303 See Other on POST delete, got %d", recPost.Code)
	}
	if mockRepo.deleted != "synapse" {
		t.Errorf("Expected device 'synapse' to be deleted, got %q", mockRepo.deleted)
	}
}

func TestDeviceHandler_ServePage(t *testing.T) {
	mockRepo := &mockDeviceRepoForHandler{
		devices: []domain.Device{
			{Name: "Synapse", Identifier: "synapse", Enabled: true, LastSeen: time.Now()},
		},
	}
	handler := NewDeviceHandler(mockRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rec := httptest.NewRecorder()

	handler.ServePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "synapse") {
		t.Errorf("Expected rendered page to contain 'synapse', got %s", body)
	}
}

func TestDeviceHandler_ServePage_RBACControls(t *testing.T) {
	mockRepo := &mockDeviceRepoForHandler{
		devices: []domain.Device{
			{Name: "Synapse", Identifier: "synapse", Enabled: true, LastSeen: time.Now()},
			{Name: "Carina Pending", Identifier: "carina-pending", Status: "PENDING", ApiKey: "key-123"},
		},
	}
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	_ = secManager.EnsureSuperuser()
	_, _ = secManager.RegisterWithRole("op_line1", "Operator#Pass2026", domain.RoleOperator)

	authHandler := NewAuthHandler(secManager)
	mw := NewAuthMiddleware(authHandler)
	handler := mw.Wrap(http.HandlerFunc(NewDeviceHandler(mockRepo, nil).ServePage))

	// 1. Admin sees active delete and cancel-pairing forms
	adminToken := secManager.CreateSession(security.SuperuserUsername, domain.RoleAdmin)
	reqAdmin := httptest.NewRequest(http.MethodGet, "/devices", nil)
	reqAdmin.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminToken})
	recAdmin := httptest.NewRecorder()
	handler.ServeHTTP(recAdmin, reqAdmin)

	if recAdmin.Code != http.StatusOK {
		t.Fatalf("Expected 200 for admin /devices, got %d", recAdmin.Code)
	}
	adminBody := recAdmin.Body.String()
	if !strings.Contains(adminBody, `action="/devices/delete"`) || !strings.Contains(adminBody, "dev_pending_cancel") {
		t.Errorf("Expected admin /devices page to contain delete and cancel-pairing forms")
	}

	// 2. Operator sees devices in read-only mode without any /devices/delete forms
	opToken := secManager.CreateSession("op_line1", domain.RoleOperator)
	reqOp := httptest.NewRequest(http.MethodGet, "/devices", nil)
	reqOp.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opToken})
	recOp := httptest.NewRecorder()
	handler.ServeHTTP(recOp, reqOp)

	if recOp.Code != http.StatusOK {
		t.Fatalf("Expected 200 for operator /devices, got %d", recOp.Code)
	}
	opBody := recOp.Body.String()
	if strings.Contains(opBody, `action="/devices/delete"`) {
		t.Errorf("Operator /devices page must NOT render /devices/delete form")
	}
	if !strings.Contains(opBody, "synapse") || !strings.Contains(opBody, "carina-pending") {
		t.Errorf("Operator /devices page should still list active and pending devices")
	}
	if !strings.Contains(opBody, "rbac_operator_readonly_notice") {
		t.Errorf("Expected operator read-only notice banner on /devices")
	}
}

func TestDeviceHandler_CRUD_KeyRotation_And_DashboardSummary(t *testing.T) {
	mockRepo := &mockDeviceRepoForHandler{
		devices: []domain.Device{
			{Name: "Existing Node", Identifier: "node-1", Status: "ACTIVE", ApiKey: "nox_live_hidden_secret", Enabled: true, LastSeen: time.Now()},
		},
	}
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	_ = secManager.EnsureSuperuser()
	_, _ = secManager.RegisterWithRole("op_watch", "Operator#Pass2026", domain.RoleOperator)

	server := NewServer("", mockRepo, &mockTelemetryRepoForServer{}, &mockContactRepo{}, nil, nil, nil, secManager, nil, nil, nil)
	router := server.Handler()

	adminCookie := &http.Cookie{Name: sessionCookieName, Value: secManager.CreateSession(security.SuperuserUsername, domain.RoleAdmin)}
	opCookie := &http.Cookie{Name: sessionCookieName, Value: secManager.CreateSession("op_watch", domain.RoleOperator)}

	// 1. /devices HTML must NOT leak ACTIVE device's api_key
	reqPage := httptest.NewRequest(http.MethodGet, "/devices", nil)
	reqPage.AddCookie(adminCookie)
	recPage := httptest.NewRecorder()
	router.ServeHTTP(recPage, reqPage)
	if recPage.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /devices, got %d", recPage.Code)
	}
	if strings.Contains(recPage.Body.String(), "nox_live_hidden_secret") {
		t.Fatal("SECURITY VIOLATION: active device api_key leaked into /devices HTML")
	}

	// 2. Admin creates device -> returns one-time api_key; Operator gets 403
	createForm := url.Values{"identifier": {"node-2"}, "name": {"Second Node"}}
	reqOpCreate := httptest.NewRequest(http.MethodPost, "/devices/create", strings.NewReader(createForm.Encode()))
	reqOpCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOpCreate.Header.Set("Accept", "application/json")
	reqOpCreate.AddCookie(opCookie)
	recOpCreate := httptest.NewRecorder()
	router.ServeHTTP(recOpCreate, reqOpCreate)
	if recOpCreate.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 for operator /devices/create, got %d", recOpCreate.Code)
	}

	reqAdminCreate := httptest.NewRequest(http.MethodPost, "/devices/create", strings.NewReader(createForm.Encode()))
	reqAdminCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAdminCreate.Header.Set("Accept", "application/json")
	reqAdminCreate.AddCookie(adminCookie)
	recAdminCreate := httptest.NewRecorder()
	router.ServeHTTP(recAdminCreate, reqAdminCreate)
	if recAdminCreate.Code != http.StatusOK || !strings.Contains(recAdminCreate.Body.String(), "nox_live_") {
		t.Fatalf("Expected 200 with generated nox_live_ key, got code=%d body=%s", recAdminCreate.Code, recAdminCreate.Body.String())
	}

	// 3. Rename, pause (toggle enabled=false), and rotate key
	updateForm := url.Values{"id": {"node-1"}, "name": {"Node One Renamed"}}
	reqUpdate := httptest.NewRequest(http.MethodPost, "/devices/update", strings.NewReader(updateForm.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUpdate.Header.Set("Accept", "application/json")
	reqUpdate.AddCookie(adminCookie)
	recUpdate := httptest.NewRecorder()
	router.ServeHTTP(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /devices/update, got %d", recUpdate.Code)
	}

	toggleForm := url.Values{"id": {"node-1"}, "enabled": {"false"}}
	reqToggle := httptest.NewRequest(http.MethodPost, "/devices/toggle", strings.NewReader(toggleForm.Encode()))
	reqToggle.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqToggle.Header.Set("Accept", "application/json")
	reqToggle.AddCookie(adminCookie)
	recToggle := httptest.NewRecorder()
	router.ServeHTTP(recToggle, reqToggle)
	if recToggle.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /devices/toggle, got %d", recToggle.Code)
	}

	rotateForm := url.Values{"id": {"node-1"}}
	reqRotate := httptest.NewRequest(http.MethodPost, "/devices/rotate-key", strings.NewReader(rotateForm.Encode()))
	reqRotate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqRotate.Header.Set("Accept", "application/json")
	reqRotate.AddCookie(adminCookie)
	recRotate := httptest.NewRecorder()
	router.ServeHTTP(recRotate, reqRotate)
	if recRotate.Code != http.StatusOK || !strings.Contains(recRotate.Body.String(), "nox_live_") {
		t.Fatalf("Expected 200 with rotated api_key, got code=%d body=%s", recRotate.Code, recRotate.Body.String())
	}

	// 4. Dashboard summary reflects PAUSED vs WAITING status cleanly for Operator
	reqSummary := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
	reqSummary.AddCookie(opCookie)
	recSummary := httptest.NewRecorder()
	router.ServeHTTP(recSummary, reqSummary)
	if recSummary.Code != http.StatusOK {
		t.Fatalf("Expected 200 on /api/dashboard/summary, got %d", recSummary.Code)
	}
	summaryBody := recSummary.Body.String()
	if !strings.Contains(summaryBody, `"status":"PAUSED"`) || !strings.Contains(summaryBody, `"status":"WAITING"`) {
		t.Errorf("Expected PAUSED and WAITING statuses in dashboard summary, got: %s", summaryBody)
	}
}

func TestDashboardHandler_ServePage_LifecycleStatusBadges(t *testing.T) {
	now := time.Now()
	repo := &mockDeviceRepoForHandler{
		devices: []domain.Device{
			{ID: 1, Name: "Sensor Pending", Identifier: "dev-pending", Status: "PENDING", Enabled: false},
			{ID: 2, Name: "Sensor Paused", Identifier: "dev-paused", Status: "ACTIVE", Enabled: false, LastSeen: now.Add(-1 * time.Minute)},
			{ID: 3, Name: "Sensor Waiting", Identifier: "dev-waiting", Status: "ACTIVE", Enabled: true},
			{ID: 4, Name: "Sensor Offline", Identifier: "dev-offline", Status: "ACTIVE", Enabled: true, LastSeen: now.Add(-10 * time.Minute)},
			{ID: 5, Name: "Sensor Online", Identifier: "dev-online", Status: "ACTIVE", Enabled: true, LastSeen: now.Add(-1 * time.Minute)},
		},
	}

	handler := NewDashboardHandler(repo, &mockTelemetryRepoForServer{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 rendering dashboard page, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	expectedTokens := []string{
		"Sensor Pending", "PENDING", "bg-warning text-dark", "table-warning",
		"Sensor Paused", "PAUSED", "bg-secondary", "table-secondary",
		"Sensor Waiting", "WAITING", "bg-info text-dark", "table-dark",
		"Sensor Offline", "OFFLINE", "bg-danger", "table-danger",
		"Sensor Online", "ONLINE", "bg-success", "table-success",
	}
	for _, token := range expectedTokens {
		if !strings.Contains(body, token) {
			t.Errorf("Expected rendered dashboard HTML to contain %q", token)
		}
	}
}
