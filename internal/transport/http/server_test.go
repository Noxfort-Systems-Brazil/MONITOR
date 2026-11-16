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
// File: internal/transport/http/server_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

type mockTelemetryRepoForServer struct{}

func (m *mockTelemetryRepoForServer) SaveEvent(_ string, _ *domain.IncomingEvent) error {
	return nil
}

func (m *mockTelemetryRepoForServer) GetRecentIncidents(_ int) ([]domain.IncomingEvent, error) {
	return nil, nil
}

func TestAuthMiddleware_UnauthenticatedAccess(t *testing.T) {
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	server := NewServer("", nil, nil, nil, nil, nil, nil, secManager, nil, nil, nil)
	handler := server.Handler()

	// 1. Unauthenticated request to "/" must serve the login page directly (status 200)
	// without any intermediate 303 redirect that would show "See Other" in desktop webviews.
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on unauthenticated GET /, got %d", recRoot.Code)
	}
	bodyRoot := recRoot.Body.String()
	if !strings.Contains(bodyRoot, "loginForm") && !strings.Contains(bodyRoot, "Entrar") {
		t.Errorf("Expected login page content on GET /, got: %s", bodyRoot)
	}

	// 2. Unauthenticated request to protected route (e.g. "/devices") must redirect to /login
	// with smart Redirect containing auto-navigation script.
	reqDevices := httptest.NewRequest(http.MethodGet, "/devices", nil)
	recDevices := httptest.NewRecorder()
	handler.ServeHTTP(recDevices, reqDevices)

	if recDevices.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303 on unauthenticated GET /devices, got %d", recDevices.Code)
	}
	if loc := recDevices.Header().Get("Location"); loc != "/login" {
		t.Errorf("Expected Location '/login', got %q", loc)
	}
	bodyDevices := recDevices.Body.String()
	if !strings.Contains(bodyDevices, `window.location.replace("/login")`) {
		t.Errorf("Expected auto-redirect script in body, got: %s", bodyDevices)
	}

	// 3. Unauthenticated request to "/api/..." must return 401 Unauthorized
	reqAPI := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	recAPI := httptest.NewRecorder()
	handler.ServeHTTP(recAPI, reqAPI)

	if recAPI.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on unauthenticated /api/..., got %d", recAPI.Code)
	}

	// 4. Unauthenticated request to /register must redirect to /login
	reqRegister := httptest.NewRequest(http.MethodGet, "/register", nil)
	recRegister := httptest.NewRecorder()
	handler.ServeHTTP(recRegister, reqRegister)
	if recRegister.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303 on unauthenticated GET /register, got %d", recRegister.Code)
	}

	// 5. Unauthenticated request to /api/users/create must return 401 Unauthorized
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", nil)
	recCreate := httptest.NewRecorder()
	handler.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on unauthenticated POST /api/users/create, got %d", recCreate.Code)
	}
}

func TestExternalIngestionHandler_BlocksBrowserAccess(t *testing.T) {
	server := NewServer("", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ingestionHandler := server.ExternalIngestionHandler()

	// 1. Browser GET request to root "/" must return 403 Forbidden
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 Forbidden on browser GET /, got %d", recRoot.Code)
	}
	bodyRoot := recRoot.Body.String()
	if !strings.Contains(bodyRoot, "Acesso via navegador desativado") {
		t.Errorf("Expected blocked message in body, got: %s", bodyRoot)
	}

	// 2. Browser GET request to "/login" must return 403 Forbidden
	reqLogin := httptest.NewRequest(http.MethodGet, "/login", nil)
	recLogin := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recLogin, reqLogin)

	if recLogin.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 Forbidden on browser GET /login, got %d", recLogin.Code)
	}

	// 3. Browser GET request to "/devices" must return 403 Forbidden
	reqDevices := httptest.NewRequest(http.MethodGet, "/devices", nil)
	recDevices := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recDevices, reqDevices)

	if recDevices.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 Forbidden on browser GET /devices, got %d", recDevices.Code)
	}

	// 4. JSON request to blocked path must return 403 Forbidden JSON
	reqJSON := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	reqJSON.Header.Set("Accept", "application/json")
	recJSON := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recJSON, reqJSON)

	if !strings.Contains(recJSON.Body.String(), "Forbidden") {
		t.Errorf("Expected Forbidden JSON response, got: %s", recJSON.Body.String())
	}

	// 5. Public Observability probes (/healthz, /metrics) must NOT be blocked with 403
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK on GET /healthz, got %d", recHealth.Code)
	}

	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	ingestionHandler.ServeHTTP(recMetrics, reqMetrics)
	if recMetrics.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK on GET /metrics, got %d", recMetrics.Code)
	}
}

func TestAuthMiddleware_OperatorRBACAndDesktopFriendly403(t *testing.T) {
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	if err := secManager.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	if _, err := secManager.RegisterWithRole("op_user", "Operator#Pass2026", domain.RoleOperator); err != nil {
		t.Fatalf("Failed to register operator: %v", err)
	}

	opToken := secManager.CreateSession("op_user", domain.RoleOperator)
	adminToken := secManager.CreateSession(security.SuperuserUsername, domain.RoleAdmin)
	opCookie := &http.Cookie{Name: sessionCookieName, Value: opToken}
	adminCookie := &http.Cookie{Name: sessionCookieName, Value: adminToken}

	server := NewServer(
		"",
		&mockDeviceRepoForHandler{},
		&mockTelemetryRepoForServer{},
		&mockContactRepo{},
		nil,
		nil,
		nil,
		secManager,
		nil,
		nil,
		nil,
	)
	handler := server.DesktopHandler()

	// 1. HTML mutating/admin requests from OPERATOR must render the Monitor HTML 403 shell (never raw text/plain)
	htmlForbiddenCases := []struct {
		method   string
		path     string
		wantBack string
	}{
		{http.MethodPost, "/devices/delete", `href="/devices"`},
		{http.MethodPost, "/contacts/delete", `href="/contacts"`},
		{http.MethodPost, "/settings/save", `href="/settings"`},
		{http.MethodGet, "/users", `href="/"`},
	}

	for _, tc := range htmlForbiddenCases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("id=1"))
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.AddCookie(opCookie)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s expected status 403, got %d", tc.method, tc.path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("%s %s expected text/html Content-Type, got %q", tc.method, tc.path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "forbidden_title") || !strings.Contains(body, `href="/"`) || !strings.Contains(body, tc.wantBack) {
			t.Errorf("%s %s expected in-app Monitor 403 template with navigation (%s), got: %s", tc.method, tc.path, tc.wantBack, body)
		}
	}

	// 2. API / JSON mutating requests from OPERATOR must return structured JSON 403
	apiForbiddenPaths := []string{
		"/api/users/delete",
		"/api/tunnel/save",
		"/api/settings/database/save",
	}
	for _, path := range apiForbiddenPaths {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Accept", "application/json")
		req.AddCookie(opCookie)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("POST %s expected status 403, got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("POST %s expected application/json Content-Type, got %q", path, ct)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("POST %s expected valid JSON error payload, decode error: %v (body=%s)", path, err, rec.Body.String())
		}
		if payload["success"] != false || payload["error"] == "" || payload["message"] == "" {
			t.Errorf("POST %s unexpected JSON 403 payload: %+v", path, payload)
		}
	}

	// 3. Settings AJAX test endpoints return non-HTML 403 JSON when requested with Accept: application/json
	reqTestSMTP := httptest.NewRequest(http.MethodPost, "/settings/test", nil)
	reqTestSMTP.Header.Set("Accept", "application/json")
	reqTestSMTP.AddCookie(opCookie)
	recTestSMTP := httptest.NewRecorder()
	handler.ServeHTTP(recTestSMTP, reqTestSMTP)
	if recTestSMTP.Code != http.StatusForbidden || !strings.Contains(recTestSMTP.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Expected 403 JSON on /settings/test for operator, got code=%d ct=%q", recTestSMTP.Code, recTestSMTP.Header().Get("Content-Type"))
	}

	// 4. Navbar for OPERATOR must hide /users and display read-only indicator, whereas ADMIN keeps /users
	reqOpDash := httptest.NewRequest(http.MethodGet, "/", nil)
	reqOpDash.AddCookie(opCookie)
	recOpDash := httptest.NewRecorder()
	handler.ServeHTTP(recOpDash, reqOpDash)
	if recOpDash.Code != http.StatusOK {
		t.Fatalf("Expected 200 on operator GET /, got %d", recOpDash.Code)
	}
	opBody := recOpDash.Body.String()
	if strings.Contains(opBody, `href="/users"`) {
		t.Errorf("Operator navbar should not render href=\"/users\"")
	}
	if !strings.Contains(opBody, "op_user") || !strings.Contains(opBody, "rbac_badge_readonly") {
		t.Errorf("Expected operator identity and read-only badge in navbar")
	}

	reqAdminDash := httptest.NewRequest(http.MethodGet, "/", nil)
	reqAdminDash.AddCookie(adminCookie)
	recAdminDash := httptest.NewRecorder()
	handler.ServeHTTP(recAdminDash, reqAdminDash)
	if recAdminDash.Code != http.StatusOK {
		t.Fatalf("Expected 200 on admin GET /, got %d", recAdminDash.Code)
	}
	if !strings.Contains(recAdminDash.Body.String(), `href="/users"`) {
		t.Errorf("Admin navbar should render href=\"/users\"")
	}
}

