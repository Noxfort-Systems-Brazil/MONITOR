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
// File: internal/monitor/alerts_test.go
// Author: Gabriel Moraes
// Date: October 2026

package monitor

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"sync"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

type mockContactRepoForAlerts struct {
	contacts []domain.Contact
	err      error
}

func (m *mockContactRepoForAlerts) GetAllContacts() ([]domain.Contact, error) {
	return m.contacts, m.err
}
func (m *mockContactRepoForAlerts) CreateContact(c *domain.Contact) error { return nil }
func (m *mockContactRepoForAlerts) UpdateContact(c *domain.Contact) error { return nil }
func (m *mockContactRepoForAlerts) DeleteContact(id int64) error         { return nil }

type mockSettingsRepoForAlerts struct {
	settings *domain.Settings
	err      error
}

func (m *mockSettingsRepoForAlerts) GetSettings() (*domain.Settings, error) {
	return m.settings, m.err
}
func (m *mockSettingsRepoForAlerts) SaveSettings(s *domain.Settings) error { return nil }
func (m *mockSettingsRepoForAlerts) GetSMTPSettings() (*domain.SMTPSettings, error) {
	return nil, nil
}
func (m *mockSettingsRepoForAlerts) SaveSMTPSettings(s *domain.SMTPSettings) error {
	return nil
}

type mockNotificationChannel struct {
	name      string
	mu        sync.Mutex
	sentCalls int
}

func (m *mockNotificationChannel) Name() string {
	return m.name
}

func (m *mockNotificationChannel) Recipient(contact *domain.Contact) string {
	if contact == nil {
		return ""
	}
	return contact.Email
}

func (m *mockNotificationChannel) Send(settings *domain.Settings, contact *domain.Contact, id string, event *domain.IncomingEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentCalls++
	return nil
}

func (m *mockNotificationChannel) getSentCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentCalls
}

func TestAlertService_TriggerAlert_EndToEnd(t *testing.T) {
	cRepo := &mockContactRepoForAlerts{
		contacts: []domain.Contact{
			{ID: 1, Name: "Admin User", Role: "admin", Enabled: true},
			{ID: 2, Name: "Disabled User", Role: "admin", Enabled: false},
			{ID: 3, Name: "Hardware Tech", Role: "technician", Enabled: true},
		},
	}

	sRepo := &mockSettingsRepoForAlerts{
		settings: &domain.Settings{
			Enabled:          true,
			TelegramBotToken: "token123",
		},
	}

	mockChan1 := &mockNotificationChannel{name: "mock1"}
	mockChan2 := &mockNotificationChannel{name: "mock2"}

	service := NewAlertServiceWithChannels(cRepo, sRepo, mockChan1, mockChan2)

	// 1. Hardware Critical alert: Admin and Hardware Tech are eligible (2 contacts x 2 channels = 4 calls)
	critEvent := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Main Pump RPM dropped below threshold",
		OccurredAt: time.Now(),
	}

	service.TriggerAlert("PUMP-01", critEvent)

	time.Sleep(50 * time.Millisecond)

	if mockChan1.getSentCalls() != 2 {
		t.Errorf("Expected 2 dispatches on mockChan1, got %d", mockChan1.getSentCalls())
	}
	if mockChan2.getSentCalls() != 2 {
		t.Errorf("Expected 2 dispatches on mockChan2, got %d", mockChan2.getSentCalls())
	}

	// 2. Info alert: must be completely ignored (0 additional calls)
	callsBefore := mockChan1.getSentCalls()
	infoEvent := &domain.IncomingEvent{
		Level:      domain.LevelInfo,
		Category:   domain.CategoryHardware,
		Message:    "Routine heartbeat",
		OccurredAt: time.Now(),
	}

	service.TriggerAlert("PUMP-01", infoEvent)
	time.Sleep(30 * time.Millisecond)

	if mockChan1.getSentCalls() != callsBefore {
		t.Errorf("INFO alert should not trigger any dispatches")
	}
}

func TestChannelTester_TestConnection(t *testing.T) {
	// 1. Without email channel (nil)
	testerNilChan := NewChannelTester(nil, nil)
	if err := testerNilChan.TestConnection(&domain.Settings{}, "test@noxfort.com"); err != nil {
		t.Errorf("Expected nil when email channel is nil, got: %v", err)
	}

	// 2. With mockable email channel
	var sentTo string
	emailChan := NewEmailChannelWithSender(func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		if len(to) > 0 {
			sentTo = to[0]
		}
		if sentTo == "fail@noxfort.com" {
			return errors.New("smtp down")
		}
		return nil
	})

	tester := NewChannelTester(emailChan, nil)

	// Success test
	settings := &domain.Settings{SMTPHost: "smtp.local", SMTPPort: 587, SMTPFrom: "alert@local"}
	if err := tester.TestConnection(settings, "admin@noxfort.com"); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if sentTo != "admin@noxfort.com" {
		t.Errorf("Expected recipient 'admin@noxfort.com', got: %s", sentTo)
	}

	// Failure test
	if err := tester.TestConnection(settings, "fail@noxfort.com"); err == nil {
		t.Errorf("Expected error on failing recipient, got nil")
	}
}

func TestChannelTester_TestTelegramConnection(t *testing.T) {
	// 1. Without telegram channel (nil)
	testerNilChan := NewChannelTester(nil, nil)
	if err := testerNilChan.TestTelegramConnection("tok", "chat123"); err != nil {
		t.Errorf("Expected nil when telegram channel is nil, got: %v", err)
	}

	// 2. With telegram channel (using dummy mock URL via endpoint)
	tgChan := NewTelegramChannelWithEndpoint(nil, "http://127.0.0.1:0")
	tester := NewChannelTester(nil, tgChan)

	// Connection error expected on closed local port
	err := tester.TestTelegramConnection("tok", "chat123")
	if err == nil {
		t.Errorf("Expected connection error on dummy address, got nil")
	}
}

func TestAlertService_TriggerAlert_RepositoryErrors(t *testing.T) {
	mockChan := &mockNotificationChannel{name: "mock"}
	event := &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Critical failure",
		OccurredAt: time.Now(),
	}

	// 1. Error fetching contacts
	serviceErrContacts := NewAlertServiceWithChannels(
		&mockContactRepoForAlerts{err: errors.New("database locked")},
		&mockSettingsRepoForAlerts{settings: &domain.Settings{Enabled: true}},
		mockChan,
	)
	serviceErrContacts.TriggerAlert("ROBOT-01", event)
	time.Sleep(20 * time.Millisecond)
	if mockChan.getSentCalls() != 0 {
		t.Errorf("Expected no dispatches when contacts query fails")
	}

	// 2. Error fetching settings
	serviceErrSettings := NewAlertServiceWithChannels(
		&mockContactRepoForAlerts{contacts: []domain.Contact{{Role: "admin", Enabled: true}}},
		&mockSettingsRepoForAlerts{err: errors.New("settings corrupt")},
		mockChan,
	)
	serviceErrSettings.TriggerAlert("ROBOT-01", event)
	time.Sleep(20 * time.Millisecond)
	if mockChan.getSentCalls() != 0 {
		t.Errorf("Expected no dispatches when settings query fails")
	}

	// 3. NewAlertService default constructor test
	serviceDefault := NewAlertService(&mockContactRepoForAlerts{}, &mockSettingsRepoForAlerts{})
	if serviceDefault == nil {
		t.Fatalf("Expected non-nil default AlertService")
	}
}

type flakyNotificationChannel struct {
	mu       sync.Mutex
	attempts int
	failNTimes int
}

func (f *flakyNotificationChannel) Name() string { return "flaky" }
func (f *flakyNotificationChannel) Recipient(c *domain.Contact) string {
	if c == nil {
		return ""
	}
	return c.Email
}
func (f *flakyNotificationChannel) Send(_ *domain.Settings, _ *domain.Contact, _ string, _ *domain.IncomingEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.attempts <= f.failNTimes {
		return errors.New("temporary network timeout")
	}
	return nil
}

type recordingAuditRepo struct {
	mu        sync.Mutex
	alertLogs []domain.AlertDispatchLog
}

func (r *recordingAuditRepo) SaveSecurityAuditLog(_ *domain.SecurityAuditLog) error { return nil }
func (r *recordingAuditRepo) GetRecentSecurityAuditLogs(_ int) ([]domain.SecurityAuditLog, error) {
	return nil, nil
}
func (r *recordingAuditRepo) SaveAlertDispatchLog(l *domain.AlertDispatchLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l != nil {
		r.alertLogs = append(r.alertLogs, *l)
	}
	return nil
}
func (r *recordingAuditRepo) GetRecentAlertDispatchLogs(_ int) ([]domain.AlertDispatchLog, error) {
	return nil, nil
}
func (r *recordingAuditRepo) SaveDeviceStateTransition(_ *domain.DeviceStateTransition) error {
	return nil
}
func (r *recordingAuditRepo) GetRecentDeviceStateTransitions(_ int) ([]domain.DeviceStateTransition, error) {
	return nil, nil
}

func TestAlertService_RetryShutdownAndTelemetryLink(t *testing.T) {
	flaky := &flakyNotificationChannel{failNTimes: 2}
	audit := &recordingAuditRepo{}

	svc := NewAlertServiceWithChannels(
		&mockContactRepoForAlerts{contacts: []domain.Contact{{ID: 1, Name: "OpsAdmin", Email: "ops@noxfort.com", Role: "admin", Enabled: true}}},
		&mockSettingsRepoForAlerts{settings: &domain.Settings{Enabled: true}},
		flaky,
	)
	svc.SetAuditRepository(audit)
	svc.SetRetryDelays(1*time.Millisecond, 1*time.Millisecond)

	telID := int64(4242)
	svc.TriggerAlertWithTelemetryID("NODE-9", &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Transient failure recovery",
		OccurredAt: time.Now(),
	}, &telID)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := svc.Shutdown(ctx); err != nil {
		t.Fatalf("Expected graceful Shutdown before timeout, got %v", err)
	}

	flaky.mu.Lock()
	attempts := flaky.attempts
	flaky.mu.Unlock()
	if attempts != 3 {
		t.Fatalf("Expected 3 total dispatch attempts (1 + 2 retries), got %d", attempts)
	}

	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.alertLogs) != 1 || audit.alertLogs[0].Status != "SENT" {
		t.Fatalf("Expected 1 SENT alert dispatch log, got %+v", audit.alertLogs)
	}
	if audit.alertLogs[0].TelemetryID == nil || *audit.alertLogs[0].TelemetryID != 4242 {
		t.Fatalf("Expected TelemetryID=4242 linked in alert dispatch log, got %+v", audit.alertLogs[0].TelemetryID)
	}
}

func TestEmailChannel_SanitizesCRLFHeaders(t *testing.T) {
	var capturedMsg string
	emailChan := NewEmailChannelWithSender(func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
		capturedMsg = string(msg)
		return nil
	})
	settings := &domain.Settings{
		SMTPHost: "smtp.local",
		SMTPPort: 587,
		SMTPFrom: "alerts@local\r\nBcc: evil@attacker.com",
		Enabled:  true,
	}
	contact := &domain.Contact{Email: "ops@local\r\nX-Inject: bad"}
	if err := emailChan.Send(settings, contact, "PLC-1", &domain.IncomingEvent{
		Level:      domain.LevelCritical,
		Category:   domain.CategoryHardware,
		Message:    "Overheat\r\nBcc: evil2@attacker.com",
		OccurredAt: time.Now(),
	}); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	headerSection := strings.SplitN(capturedMsg, "\r\n\r\n", 2)[0]
	for _, line := range strings.Split(headerSection, "\r\n") {
		lowerLine := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lowerLine, "bcc:") || strings.HasPrefix(lowerLine, "x-inject:") {
			t.Fatalf("SECURITY VIOLATION: CRLF SMTP header injection escaped onto new header line:\n%s", headerSection)
		}
	}
	if strings.Contains(headerSection, "\nX-Inject:") || strings.Contains(headerSection, "\nBcc:") {
		t.Fatalf("SECURITY VIOLATION: unexpected injected newline in header section:\n%s", headerSection)
	}
}
