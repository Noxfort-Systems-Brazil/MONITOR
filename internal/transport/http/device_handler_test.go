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
