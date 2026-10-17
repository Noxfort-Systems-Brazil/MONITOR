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
