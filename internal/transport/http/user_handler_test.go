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
// File: internal/transport/http/user_handler_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

type mockUserRepoForHandler struct {
	users map[string]*domain.User
}

func (m *mockUserRepoForHandler) GetByUsername(username string) (*domain.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockUserRepoForHandler) List() ([]*domain.User, error) {
	var l []*domain.User
	for _, u := range m.users {
		l = append(l, u)
	}
	return l, nil
}

func (m *mockUserRepoForHandler) Create(u *domain.User) error {
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepoForHandler) UpdatePasswordHash(username, passwordHash string) error {
	if u, ok := m.users[username]; ok {
		u.PasswordHash = passwordHash
		return nil
	}
	return nil
}

func (m *mockUserRepoForHandler) UpdateRole(username, role string) error {
	if u, ok := m.users[username]; ok {
		u.Role = role
		return nil
	}
	return nil
}

func (m *mockUserRepoForHandler) Delete(id int64) error {
	return nil
}

func (m *mockUserRepoForHandler) DeleteByUsername(username string) error {
	delete(m.users, username)
	return nil
}

func (m *mockUserRepoForHandler) CountAdmins() (int, error) {
	count := 0
	for _, u := range m.users {
		if u.Role == domain.RoleAdmin {
			count++
		}
	}
	return count, nil
}

func TestUserHandler_CRUDAndAuthorization(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	authHandler := NewAuthHandler(sm)
	userHandler := NewUserHandler(sm, sm)

	// 1. Unauthenticated attempt to register must be rejected (403)
	formOp := url.Values{}
	formOp.Set("username", "plant_op")
	formOp.Set("password", "Operator#Pass2026")

	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqUnauth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrUnauth := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusForbidden {
		t.Fatalf("Expected unauthenticated user creation to return 403, got %d", rrUnauth.Code)
	}

	// 2. Login with configured Superuser credentials via AuthHandler
	loginForm := url.Values{}
	loginForm.Set("username", security.SuperuserUsername)
	loginForm.Set("password", security.SuperuserPassword)

	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginForm.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrLogin := httptest.NewRecorder()

	authHandler.HandleLogin(rrLogin, reqLogin)
	if rrLogin.Code != http.StatusOK {
		t.Fatalf("Expected superuser login status 200, got %d. Body: %s", rrLogin.Code, rrLogin.Body.String())
	}

	// Extract admin session cookie
	cookies := rrLogin.Result().Cookies()
	var adminCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			adminCookie = c
			break
		}
	}
	if adminCookie == nil {
		t.Fatalf("Expected session cookie to be set upon superuser login")
	}

	// 3. Authenticated Admin creates new Operator
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreate.AddCookie(adminCookie)
	rrCreate := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusOK {
		t.Fatalf("Expected operator creation to return 200, got %d. Body: %s", rrCreate.Code, rrCreate.Body.String())
	}

	var resOp map[string]interface{}
	_ = json.NewDecoder(rrCreate.Body).Decode(&resOp)
	if resOp["role"] != domain.RoleOperator {
		t.Fatalf("Expected created user to have role %s, got %v", domain.RoleOperator, resOp["role"])
	}

	// 4. Operator logs in successfully via AuthHandler
	opLoginForm := url.Values{}
	opLoginForm.Set("username", "plant_op")
	opLoginForm.Set("password", "Operator#Pass2026")

	reqOpLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(opLoginForm.Encode()))
	reqOpLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrOpLogin := httptest.NewRecorder()

	authHandler.HandleLogin(rrOpLogin, reqOpLogin)
	if rrOpLogin.Code != http.StatusOK {
		t.Fatalf("Expected operator login 200, got %d", rrOpLogin.Code)
	}

	// Extract operator session cookie
	var opCookie *http.Cookie
	for _, c := range rrOpLogin.Result().Cookies() {
		if c.Name == sessionCookieName {
			opCookie = c
			break
		}
	}
	if opCookie == nil {
		t.Fatalf("Expected operator session cookie")
	}

	// Operator cannot create users (must be 403)
	reqOpCreate := httptest.NewRequest(http.MethodPost, "/api/users/create", strings.NewReader(formOp.Encode()))
	reqOpCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOpCreate.AddCookie(opCookie)
	rrOpCreate := httptest.NewRecorder()

	userHandler.HandleCreateUser(rrOpCreate, reqOpCreate)
	if rrOpCreate.Code != http.StatusForbidden {
		t.Fatalf("Expected operator user creation to return 403, got %d", rrOpCreate.Code)
	}

	// 5. Admin lists users
	reqList := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqList.AddCookie(adminCookie)
	rrList := httptest.NewRecorder()

	userHandler.HandleList(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("Expected list status 200, got %d", rrList.Code)
	}

	// 6. Admin deletes operator
	delForm := url.Values{}
	delForm.Set("username", "plant_op")

	reqDel := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(delForm.Encode()))
	reqDel.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDel.AddCookie(adminCookie)
	rrDel := httptest.NewRecorder()

	userHandler.HandleDelete(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("Expected delete status 200, got %d. Body: %s", rrDel.Code, rrDel.Body.String())
	}

	// 7. Admin tries to delete superuser (must be blocked)
	delSuperForm := url.Values{}
	delSuperForm.Set("username", security.SuperuserUsername)

	reqDelSuper := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(delSuperForm.Encode()))
	reqDelSuper.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDelSuper.AddCookie(adminCookie)
	rrDelSuper := httptest.NewRecorder()

	userHandler.HandleDelete(rrDelSuper, reqDelSuper)
	if rrDelSuper.Code != http.StatusBadRequest {
		t.Fatalf("Expected deleting superuser to return 400, got %d", rrDelSuper.Code)
	}

	// 8. Secondary admin self-deletes (allowed, terminates session)
	_, _ = sm.RegisterWithRole("admin2", "Admin2#Pass2026", domain.RoleAdmin)
	token2 := sm.CreateSession("admin2", domain.RoleAdmin)
	admin2Cookie := &http.Cookie{Name: "noxfort_session", Value: token2}

	delSelfForm := url.Values{}
	delSelfForm.Set("username", "admin2")
	reqDelSelf := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(delSelfForm.Encode()))
	reqDelSelf.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDelSelf.AddCookie(admin2Cookie)
	rrDelSelf := httptest.NewRecorder()

	userHandler.HandleDelete(rrDelSelf, reqDelSelf)
	if rrDelSelf.Code != http.StatusOK {
		t.Fatalf("Expected self-delete to return 200, got %d", rrDelSelf.Code)
	}
	if !strings.Contains(rrDelSelf.Body.String(), `"self_deleted":true`) {
		t.Fatalf("Expected self_deleted true in response, got %s", rrDelSelf.Body.String())
	}
}

func TestUserHandler_TokenInHeadersAndQuery(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	handler := NewUserHandler(sm, sm)
	token := sm.CreateSession(security.SuperuserUsername, domain.RoleAdmin)

	// 1. Validate session via Authorization Bearer header
	reqBearer := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqBearer.Header.Set("Authorization", "Bearer "+token)
	username, role, isAuth := handler.GetSessionUser(reqBearer)
	if !isAuth || username != security.SuperuserUsername || role != domain.RoleAdmin {
		t.Fatalf("Expected valid session via Bearer header, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}

	// 2. Validate session via X-Session-Token header
	reqCustom := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqCustom.Header.Set("X-Session-Token", token)
	username, role, isAuth = handler.GetSessionUser(reqCustom)
	if !isAuth || username != security.SuperuserUsername || role != domain.RoleAdmin {
		t.Fatalf("Expected valid session via X-Session-Token header, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}

	// 3. URL query param (?token=...) must be rejected to avoid leaking tokens in access logs
	reqQuery := httptest.NewRequest(http.MethodGet, "/api/users?token="+token, nil)
	username, role, isAuth = handler.GetSessionUser(reqQuery)
	if isAuth || username != "" || role != "" {
		t.Fatalf("Expected query parameter session token to be rejected, got user=%q, role=%q, isAuth=%v", username, role, isAuth)
	}
}

func TestUserHandler_DeleteErrorHeadersAndOperatorForbiddenHTML(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("Failed to initialize superuser: %v", err)
	}
	if _, err := sm.RegisterWithRole("op_view", "Operator#Pass2026", domain.RoleOperator); err != nil {
		t.Fatalf("Failed to register operator: %v", err)
	}

	handler := NewUserHandler(sm, sm)
	adminToken := sm.CreateSession(security.SuperuserUsername, domain.RoleAdmin)
	opToken := sm.CreateSession("op_view", domain.RoleOperator)

	// 1. Delete error on superuser must set Content-Type: application/json BEFORE WriteHeader
	superForm := url.Values{"username": {security.SuperuserUsername}}
	reqSuper := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(superForm.Encode()))
	reqSuper.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqSuper.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminToken})
	recSuper := httptest.NewRecorder()

	handler.HandleDelete(recSuper, reqSuper)
	if recSuper.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", recSuper.Code)
	}
	if ct := recSuper.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Expected Content-Type application/json on HandleDelete 400 response, got %q", ct)
	}
	var errPayload map[string]interface{}
	if err := json.Unmarshal(recSuper.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("Expected valid JSON error payload, got error: %v", err)
	}

	// 2. Empty username must also return JSON 400
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/users/delete", strings.NewReader(""))
	reqEmpty.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqEmpty.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminToken})
	recEmpty := httptest.NewRecorder()
	handler.HandleDelete(recEmpty, reqEmpty)
	if recEmpty.Code != http.StatusBadRequest || !strings.Contains(recEmpty.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Expected 400 application/json for empty username, got code=%d ct=%q", recEmpty.Code, recEmpty.Header().Get("Content-Type"))
	}

	// 3. Operator accessing /users ServePage receives 403 HTML shell instead of plain text
	reqPage := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqPage.Header.Set("Accept", "text/html")
	reqPage.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opToken})
	recPage := httptest.NewRecorder()
	handler.ServePage(recPage, reqPage)

	if recPage.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 on operator GET /users, got %d", recPage.Code)
	}
	if ct := recPage.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Expected text/html Content-Type on operator /users 403, got %q", ct)
	}
	if !strings.Contains(recPage.Body.String(), "forbidden_title") || !strings.Contains(recPage.Body.String(), `href="/"`) {
		t.Errorf("Expected rendered Monitor forbidden template, got: %s", recPage.Body.String())
	}
}

func TestUserHandler_PasswordRoleUnlockAndRBAC(t *testing.T) {
	repo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	sm := security.NewSecurityManager(repo)
	if err := sm.EnsureSuperuser(); err != nil {
		t.Fatalf("EnsureSuperuser failed: %v", err)
	}
	if _, err := sm.RegisterWithRole("op_shift", "Operator#Pass2026", domain.RoleOperator); err != nil {
		t.Fatalf("Register operator failed: %v", err)
	}

	server := NewServer("", nil, nil, nil, nil, nil, nil, sm, nil, nil, nil)
	router := server.Handler()

	adminTok := sm.CreateSession(security.SuperuserUsername, domain.RoleAdmin)
	opTok := sm.CreateSession("op_shift", domain.RoleOperator)

	// 1. Operator can change their OWN password when supplying valid current_password
	selfPassForm := url.Values{
		"username":         {"op_shift"},
		"current_password": {"Operator#Pass2026"},
		"new_password":     {"Operator#New2026"},
	}
	reqSelfPass := httptest.NewRequest(http.MethodPost, "/api/users/password", strings.NewReader(selfPassForm.Encode()))
	reqSelfPass.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqSelfPass.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opTok})
	recSelfPass := httptest.NewRecorder()
	router.ServeHTTP(recSelfPass, reqSelfPass)
	if recSelfPass.Code != http.StatusOK {
		t.Fatalf("Expected 200 on operator self password change, got %d: %s", recSelfPass.Code, recSelfPass.Body.String())
	}
	if _, _, valid := sm.ValidateSession(opTok); valid {
		t.Error("Expected operator's previous session to be revoked after password change")
	}

	// 2. Operator CANNOT change admin's password (403) or update roles (403)
	opNewTok := sm.CreateSession("op_shift", domain.RoleOperator)
	otherPassForm := url.Values{
		"username":     {security.SuperuserUsername},
		"new_password": {"Hacked@2026!"},
	}
	reqOtherPass := httptest.NewRequest(http.MethodPost, "/api/users/password", strings.NewReader(otherPassForm.Encode()))
	reqOtherPass.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOtherPass.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opNewTok})
	recOtherPass := httptest.NewRecorder()
	router.ServeHTTP(recOtherPass, reqOtherPass)
	if recOtherPass.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 when operator tries changing another account's password, got %d", recOtherPass.Code)
	}

	roleForm := url.Values{"username": {"op_shift"}, "role": {domain.RoleAdmin}}
	reqOpRole := httptest.NewRequest(http.MethodPost, "/api/users/role", strings.NewReader(roleForm.Encode()))
	reqOpRole.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOpRole.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opNewTok})
	recOpRole := httptest.NewRecorder()
	router.ServeHTTP(recOpRole, reqOpRole)
	if recOpRole.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 when operator tries /api/users/role, got %d", recOpRole.Code)
	}

	// 3. Admin promotes op_shift -> ADMIN and unlocks account
	reqAdminRole := httptest.NewRequest(http.MethodPost, "/api/users/role", strings.NewReader(roleForm.Encode()))
	reqAdminRole.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAdminRole.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminTok})
	recAdminRole := httptest.NewRecorder()
	router.ServeHTTP(recAdminRole, reqAdminRole)
	if recAdminRole.Code != http.StatusOK {
		t.Fatalf("Expected 200 on admin role promotion, got %d: %s", recAdminRole.Code, recAdminRole.Body.String())
	}

	unlockForm := url.Values{"username": {"op_shift"}}
	reqUnlock := httptest.NewRequest(http.MethodPost, "/api/users/unlock", strings.NewReader(unlockForm.Encode()))
	reqUnlock.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUnlock.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminTok})
	recUnlock := httptest.NewRecorder()
	router.ServeHTTP(recUnlock, reqUnlock)
	if recUnlock.Code != http.StatusOK {
		t.Fatalf("Expected 200 on admin unlock, got %d: %s", recUnlock.Code, recUnlock.Body.String())
	}
}
