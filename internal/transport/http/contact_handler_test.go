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
// File: internal/transport/http/contact_handler_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

type mockContactRepo struct {
	contacts []domain.Contact
	nextID   int64
}

func (m *mockContactRepo) GetAllContacts() ([]domain.Contact, error) {
	return m.contacts, nil
}

func (m *mockContactRepo) CreateContact(c *domain.Contact) error {
	m.nextID++
	c.ID = m.nextID
	m.contacts = append(m.contacts, *c)
	return nil
}

func (m *mockContactRepo) UpdateContact(c *domain.Contact) error {
	for i, existing := range m.contacts {
		if existing.ID == c.ID {
			m.contacts[i] = *c
			return nil
		}
	}
	return nil
}

func (m *mockContactRepo) DeleteContact(id int64) error {
	for i, existing := range m.contacts {
		if existing.ID == id {
			m.contacts = append(m.contacts[:i], m.contacts[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestContactHandler_CreateUpdateDelete(t *testing.T) {
	repo := &mockContactRepo{}
	handler := NewContactHandler(repo)

	// 1. Create
	createForm := url.Values{
		"name":             {"Bob Builder"},
		"role":             {"Programmer"},
		"email":            {"bob@noxfort.com"},
		"phone":            {"+55 11 91111-2222"},
		"telegram_chat_id": {"999888"},
		"notify_critical":  {"on"},
	}

	reqCreate := httptest.NewRequest(http.MethodPost, "/contacts/create", strings.NewReader(createForm.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recCreate := httptest.NewRecorder()

	handler.HandleCreate(recCreate, reqCreate)

	if recCreate.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recCreate.Code)
	}

	if len(repo.contacts) != 1 {
		t.Fatalf("Expected 1 contact in repo, got %d", len(repo.contacts))
	}
	if repo.contacts[0].Name != "Bob Builder" || !repo.contacts[0].NotifyCritical {
		t.Errorf("Unexpected contact: %+v", repo.contacts[0])
	}

	// 2. Update
	updateForm := url.Values{
		"id":               {"1"},
		"name":             {"Bob The Builder"},
		"role":             {"System Admin"},
		"email":            {"bob.builder@noxfort.com"},
		"phone":            {"+55 11 93333-4444"},
		"telegram_chat_id": {"112233"},
		"notify_critical":  {""},
	}

	reqUpdate := httptest.NewRequest(http.MethodPost, "/contacts/update", strings.NewReader(updateForm.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recUpdate := httptest.NewRecorder()

	handler.HandleUpdate(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recUpdate.Code)
	}

	if len(repo.contacts) != 1 {
		t.Fatalf("Expected 1 contact in repo, got %d", len(repo.contacts))
	}
	updated := repo.contacts[0]
	if updated.Name != "Bob The Builder" ||
		updated.Role != "System Admin" ||
		updated.Email != "bob.builder@noxfort.com" ||
		updated.Phone != "+55 11 93333-4444" ||
		updated.TelegramChatID != "112233" ||
		updated.NotifyCritical != false {
		t.Errorf("Unexpected updated contact: %+v", updated)
	}

	// 3. Delete
	deleteForm := url.Values{
		"id": {"1"},
	}

	reqDelete := httptest.NewRequest(http.MethodPost, "/contacts/delete", strings.NewReader(deleteForm.Encode()))
	reqDelete.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recDelete := httptest.NewRecorder()

	handler.HandleDelete(recDelete, reqDelete)

	if recDelete.Code != http.StatusSeeOther {
		t.Fatalf("Expected status 303, got %d", recDelete.Code)
	}

	if len(repo.contacts) != 0 {
		t.Fatalf("Expected 0 contacts after delete, got %d", len(repo.contacts))
	}
}

func TestContactHandler_ServePage_RBACControls(t *testing.T) {
	repo := &mockContactRepo{
		contacts: []domain.Contact{
			{ID: 1, Name: "Alice Operator", Role: "Technician", Email: "alice@noxfort.com", Enabled: true},
		},
	}
	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	_ = secManager.EnsureSuperuser()
	_, _ = secManager.RegisterWithRole("op_contacts", "Operator#Pass2026", domain.RoleOperator)

	authHandler := NewAuthHandler(secManager)
	mw := NewAuthMiddleware(authHandler)
	handler := mw.Wrap(http.HandlerFunc(NewContactHandler(repo).ServePage))

	// 1. Admin sees create, edit modal, and delete forms
	adminToken := secManager.CreateSession(security.SuperuserUsername, domain.RoleAdmin)
	reqAdmin := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	reqAdmin.AddCookie(&http.Cookie{Name: sessionCookieName, Value: adminToken})
	recAdmin := httptest.NewRecorder()
	handler.ServeHTTP(recAdmin, reqAdmin)

	if recAdmin.Code != http.StatusOK {
		t.Fatalf("Expected 200 for admin /contacts, got %d", recAdmin.Code)
	}
	adminBody := recAdmin.Body.String()
	for _, action := range []string{`action="/contacts/create"`, `action="/contacts/update"`, `action="/contacts/delete"`} {
		if !strings.Contains(adminBody, action) {
			t.Errorf("Expected admin /contacts page to contain %s", action)
		}
	}

	// 2. Operator sees directory in read-only mode without create, update, or delete controls
	opToken := secManager.CreateSession("op_contacts", domain.RoleOperator)
	reqOp := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	reqOp.AddCookie(&http.Cookie{Name: sessionCookieName, Value: opToken})
	recOp := httptest.NewRecorder()
	handler.ServeHTTP(recOp, reqOp)

	if recOp.Code != http.StatusOK {
		t.Fatalf("Expected 200 for operator /contacts, got %d", recOp.Code)
	}
	opBody := recOp.Body.String()
	for _, action := range []string{`action="/contacts/create"`, `action="/contacts/update"`, `action="/contacts/delete"`} {
		if strings.Contains(opBody, action) {
			t.Errorf("Operator /contacts page must NOT contain %s", action)
		}
	}
	if !strings.Contains(opBody, "Alice Operator") || !strings.Contains(opBody, "rbac_operator_readonly_notice") {
		t.Errorf("Expected read-only contact directory and operator notice")
	}
}
