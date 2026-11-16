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
// File: internal/transport/http/database_handler_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
	"noxfort-monitor-server/internal/storage"
)

func TestDatabaseHandler_Endpoints(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("MONITOR_DB_CONFIG_PATH", filepath.Join(tempDir, "database_config.json"))
	dbPath := filepath.Join(tempDir, "handler_test.db")

	cfg := domain.DatabaseConfig{
		Type:     "sqlite",
		FilePath: dbPath,
	}

	db, driver, err := storage.OpenConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	_ = storage.InitPostgresSchema(db, "schema_monitor") // will be ignored for sqlite or error handled
	dbManager := storage.NewDBManager(db, driver, cfg)
	auditRepo := storage.NewAuditRepository(db, driver)
	handler := NewDatabaseHandler(dbManager, auditRepo)

	// 1. GET /api/settings/database/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/settings/database/status", nil)
	recStatus := httptest.NewRecorder()
	handler.HandleStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleStatus, got %d", recStatus.Code)
	}

	var statusResp map[string]interface{}
	if err := json.NewDecoder(recStatus.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	if statusResp["status"] == nil {
		t.Errorf("Expected status object in response")
	}

	// 2. POST /api/settings/database/test (Testing SQLite)
	testBody := `{"type":"sqlite","file_path":"` + dbPath + `"}`
	reqTest := httptest.NewRequest(http.MethodPost, "/api/settings/database/test", strings.NewReader(testBody))
	reqTest.Header.Set("Content-Type", "application/json")
	recTest := httptest.NewRecorder()
	handler.HandleTest(recTest, reqTest)

	if recTest.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleTest, got %d: %s", recTest.Code, recTest.Body.String())
	}

	var testResp map[string]interface{}
	if err := json.NewDecoder(recTest.Body).Decode(&testResp); err != nil {
		t.Fatalf("Failed to decode test response: %v", err)
	}
	if testResp["success"] != true {
		t.Errorf("Expected success to be true, got %v", testResp["success"])
	}

	// 3. POST /api/settings/database/save (Saving SQLite)
	dbPath2 := filepath.Join(tempDir, "handler_test_save.db")
	saveBody := `{"type":"sqlite","file_path":"` + dbPath2 + `","migrate":false}`
	reqSave := httptest.NewRequest(http.MethodPost, "/api/settings/database/save", strings.NewReader(saveBody))
	reqSave.Header.Set("Content-Type", "application/json")
	recSave := httptest.NewRecorder()
	handler.HandleSave(recSave, reqSave)

	if recSave.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleSave, got %d: %s", recSave.Code, recSave.Body.String())
	}

	// 4. GET /server (ServePage)
	reqPage := httptest.NewRequest(http.MethodGet, "/server", nil)
	recPage := httptest.NewRecorder()
	handler.ServePage(recPage, reqPage)

	if recPage.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on ServePage, got %d: %s", recPage.Code, recPage.Body.String())
	}
	if !strings.Contains(recPage.Body.String(), "dbSettingsForm") {
		t.Errorf("Expected ServePage output to contain dbSettingsForm")
	}
}

type mockDBService struct {
	status   domain.DatabaseStatus
	cfg      domain.DatabaseConfig
	testErr  error
	switchErr error
}

func (m *mockDBService) GetStatus() domain.DatabaseStatus {
	return m.status
}

func (m *mockDBService) GetConfig() domain.DatabaseConfig {
	return m.cfg
}

func (m *mockDBService) TestConnection(cfg domain.DatabaseConfig) (domain.DatabaseStatus, error) {
	return m.status, m.testErr
}

func (m *mockDBService) Switch(newCfg domain.DatabaseConfig, migrate bool) error {
	m.cfg = newCfg
	return m.switchErr
}

type mockProvisioner struct {
	provisionErr error
	called       bool
}

func (m *mockProvisioner) ProvisionUser(host string, port int, dbname, adminUser, adminPassword, newUser, newPassword, sslmode string) error {
	m.called = true
	return m.provisionErr
}

func TestDatabaseHandler_MockService(t *testing.T) {
	mockSvc := &mockDBService{
		status: domain.DatabaseStatus{
			Connected:    true,
			Type:         "postgres",
			SchemaExists: true,
		},
		cfg: domain.DatabaseConfig{
			Type:   "postgres",
			Host:   "localhost",
			DBName: "testdb",
		},
	}
	mockProv := &mockProvisioner{}

	handler := NewDatabaseHandlerWithProvisioner(mockSvc, nil, mockProv)

	// Test Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/settings/database/status", nil)
	recStatus := httptest.NewRecorder()
	handler.HandleStatus(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", recStatus.Code)
	}

	// Test Provision User
	provBody := `{"host":"localhost","port":5432,"dbname":"testdb","new_user":"testuser","new_password":"pwd","admin_user":"postgres","admin_password":"pwd"}`
	reqProv := httptest.NewRequest(http.MethodPost, "/api/settings/database/provision-user", strings.NewReader(provBody))
	reqProv.Header.Set("Content-Type", "application/json")
	recProv := httptest.NewRecorder()
	handler.HandleProvisionUser(recProv, reqProv)

	if recProv.Code != http.StatusOK {
		t.Errorf("Expected 200 on provision-user, got %d: %s", recProv.Code, recProv.Body.String())
	}
	if !mockProv.called {
		t.Errorf("Expected provisioner to be called")
	}
}

func TestDatabaseHandler_AuditUsesSessionUsername(t *testing.T) {
	auditSpy := &spyAuditRepo{}
	mockSvc := &mockDBService{status: domain.DatabaseStatus{Connected: true, Type: "postgres"}}
	handler := NewDatabaseHandlerWithProvisioner(mockSvc, auditSpy, &mockProvisioner{})

	req := httptest.NewRequest(http.MethodPost, "/api/settings/database/save", strings.NewReader(`{"type":"sqlite","file_path":"/tmp/x.db","migrate":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.10.22:5555"
	req = withSessionContext(req, "carlos_eng", domain.RoleAdmin)
	rec := httptest.NewRecorder()

	handler.HandleSave(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 on HandleSave, got %d", rec.Code)
	}
	if len(auditSpy.secLogs) != 1 || auditSpy.secLogs[0].Username != "carlos_eng" || auditSpy.secLogs[0].IPAddress != "192.168.10.22" {
		t.Fatalf("Expected audit log attributed to carlos_eng@192.168.10.22, got %+v", auditSpy.secLogs)
	}
}

func TestDatabaseHandler_BackupDownloadRestore_And_OperatorBlocked(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("MONITOR_DB_CONFIG_PATH", filepath.Join(tempDir, "database_config.json"))
	dbPath := filepath.Join(tempDir, "live.db")
	backupDir := filepath.Join(tempDir, "backups")

	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase failed: %v", err)
	}
	defer db.Close()

	cfg := domain.DatabaseConfig{Type: "sqlite", FilePath: dbPath}
	dbm := storage.NewDBManager(db, "sqlite", cfg)
	devRepo := storage.NewDeviceRepository(db)
	dbm.RegisterRepository(devRepo)
	bm := storage.NewBackupManager(dbm, backupDir, 7)

	if _, err := devRepo.CreateManualDevice("Sensor Alpha", "alpha-1", "nox_live_alpha"); err != nil {
		t.Fatalf("CreateManualDevice failed: %v", err)
	}

	userRepo := &mockUserRepoForHandler{users: make(map[string]*domain.User)}
	secManager := security.NewSecurityManager(userRepo)
	_ = secManager.EnsureSuperuser()
	_, _ = secManager.RegisterWithRole("op_ro", "Operator#Pass2026", domain.RoleOperator)

	srv := NewServer("", devRepo, nil, nil, nil, nil, nil, secManager, nil, dbm, nil)
	srv.SetBackupService(bm)
	router := srv.Handler()

	adminCookie := &http.Cookie{Name: sessionCookieName, Value: secManager.CreateSession(security.SuperuserUsername, domain.RoleAdmin)}
	opCookie := &http.Cookie{Name: sessionCookieName, Value: secManager.CreateSession("op_ro", domain.RoleOperator)}

	// 1. Admin creates backup
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/settings/database/backup", nil)
	reqCreate.AddCookie(adminCookie)
	recCreate := httptest.NewRecorder()
	router.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusOK {
		t.Fatalf("Expected 200 on POST backup, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	var createResp struct {
		Backup domain.BackupMetadata `json:"backup"`
	}
	_ = json.Unmarshal(recCreate.Body.Bytes(), &createResp)
	if createResp.Backup.Filename == "" {
		t.Fatalf("Expected backup filename in response")
	}

	// 2. Operator must be blocked (403) from listing or downloading DB backups
	reqOpList := httptest.NewRequest(http.MethodGet, "/api/settings/database/backups", nil)
	reqOpList.AddCookie(opCookie)
	recOpList := httptest.NewRecorder()
	router.ServeHTTP(recOpList, reqOpList)
	if recOpList.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 when operator lists backups, got %d", recOpList.Code)
	}

	reqOpDL := httptest.NewRequest(http.MethodGet, "/api/settings/database/backups/download?file="+createResp.Backup.Filename, nil)
	reqOpDL.AddCookie(opCookie)
	recOpDL := httptest.NewRecorder()
	router.ServeHTTP(recOpDL, reqOpDL)
	if recOpDL.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 when operator downloads backup, got %d", recOpDL.Code)
	}

	// 3. Admin downloads backup successfully and rejects traversal
	reqAdminDL := httptest.NewRequest(http.MethodGet, "/api/settings/database/backups/download?file="+createResp.Backup.Filename, nil)
	reqAdminDL.AddCookie(adminCookie)
	recAdminDL := httptest.NewRecorder()
	router.ServeHTTP(recAdminDL, reqAdminDL)
	if recAdminDL.Code != http.StatusOK || recAdminDL.Body.Len() == 0 {
		t.Fatalf("Expected non-empty 200 download, got code=%d len=%d", recAdminDL.Code, recAdminDL.Body.Len())
	}

	reqBadDL := httptest.NewRequest(http.MethodGet, "/api/settings/database/backups/download?file=../../etc/passwd", nil)
	reqBadDL.AddCookie(adminCookie)
	recBadDL := httptest.NewRecorder()
	router.ServeHTTP(recBadDL, reqBadDL)
	if recBadDL.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 on path traversal download attempt, got %d", recBadDL.Code)
	}

	// 4. Delete device, restore backup via HTTP, verify device came back!
	_ = devRepo.DeleteDevice("alpha-1")
	if d, _ := devRepo.GetByIdentifier("alpha-1"); d != nil {
		t.Fatal("Expected alpha-1 deleted before restore test")
	}
	_ = os.Setenv("NOXFORT_KEEP_SQLITE_LOCK_CHECK", "")
	reqRestore := httptest.NewRequest(http.MethodPost, "/api/settings/database/restore", strings.NewReader(`{"filename":"`+createResp.Backup.Filename+`"}`))
	reqRestore.Header.Set("Content-Type", "application/json")
	reqRestore.AddCookie(adminCookie)
	recRestore := httptest.NewRecorder()
	router.ServeHTTP(recRestore, reqRestore)
	if recRestore.Code != http.StatusOK {
		t.Fatalf("Expected 200 on restore, got %d: %s", recRestore.Code, recRestore.Body.String())
	}
	if d, _ := devRepo.GetByIdentifier("alpha-1"); d == nil {
		t.Fatal("Expected alpha-1 restored from backup via HTTP handler")
	}
}

type spyAuditRepo struct {
	secLogs []domain.SecurityAuditLog
}

func (s *spyAuditRepo) SaveSecurityAuditLog(l *domain.SecurityAuditLog) error {
	if l != nil {
		s.secLogs = append(s.secLogs, *l)
	}
	return nil
}
func (s *spyAuditRepo) GetRecentSecurityAuditLogs(_ int) ([]domain.SecurityAuditLog, error) {
	return s.secLogs, nil
}
func (s *spyAuditRepo) SaveAlertDispatchLog(_ *domain.AlertDispatchLog) error { return nil }
func (s *spyAuditRepo) GetRecentAlertDispatchLogs(_ int) ([]domain.AlertDispatchLog, error) {
	return nil, nil
}
func (s *spyAuditRepo) SaveDeviceStateTransition(_ *domain.DeviceStateTransition) error { return nil }
func (s *spyAuditRepo) GetRecentDeviceStateTransitions(_ int) ([]domain.DeviceStateTransition, error) {
	return nil, nil
}

