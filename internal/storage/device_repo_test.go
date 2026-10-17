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
// File: internal/storage/device_repo_test.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup
}

func TestDeviceRepository_DebouncedHeartbeat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewDeviceRepository(db)
	repo.SetDebounceInterval(500 * time.Millisecond)

	now := time.Now().Truncate(time.Second)

	// 1. First heartbeat: must persist immediately
	err := repo.UpdateLastSeen("MACHINE-01", now)
	if err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	devices, err := repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}
	if devices[0].Identifier != "MACHINE-01" {
		t.Errorf("Expected identifier MACHINE-01, got %s", devices[0].Identifier)
	}

	// 2. Second heartbeat within debounce interval (100ms later)
	secondTime := now.Add(100 * time.Millisecond)
	err = repo.UpdateLastSeen("MACHINE-01", secondTime)
	if err != nil {
		t.Fatalf("Second UpdateLastSeen failed: %v", err)
	}

	// GetAllDevices should reflect secondTime thanks to the in-memory cache overlay
	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if !devices[0].LastSeen.Equal(secondTime) {
		t.Errorf("Expected LastSeen overlay %v, got %v", secondTime, devices[0].LastSeen)
	}

	// 3. Third heartbeat after debounce interval has passed
	time.Sleep(550 * time.Millisecond)
	thirdTime := secondTime.Add(600 * time.Millisecond)
	err = repo.UpdateLastSeen("MACHINE-01", thirdTime)
	if err != nil {
		t.Fatalf("Third UpdateLastSeen failed: %v", err)
	}

	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed: %v", err)
	}
	if !devices[0].LastSeen.Equal(thirdTime) {
		t.Errorf("Expected LastSeen %v, got %v", thirdTime, devices[0].LastSeen)
	}

	// 4. ForceUpdateLastSeen should always write directly
	forcedTime := thirdTime.Add(1 * time.Second)
	err = repo.ForceUpdateLastSeen("MACHINE-01", forcedTime)
	if err != nil {
		t.Fatalf("ForceUpdateLastSeen failed: %v", err)
	}

	// 5. Delete device
	err = repo.DeleteDevice("MACHINE-01")
	if err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}

	devices, err = repo.GetAllDevices()
	if err != nil {
		t.Fatalf("GetAllDevices failed after deletion: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("Expected 0 devices after deletion, got %d", len(devices))
	}
}

func TestDeviceRepository_PairingAndTokens(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewDeviceRepository(db)

	// 1. Register a pending device
	expiresAt := time.Now().Add(5 * time.Minute)
	pendingDev := &domain.Device{
		Identifier: "carina-test-01",
		Name:       "Carina Test Node",
		ApiKey:     "nox_live_test_token_12345",
		Status:     "PENDING",
		ExpiresAt:  &expiresAt,
		Enabled:    true,
	}

	if err := repo.CreateOrUpdatePending(pendingDev); err != nil {
		t.Fatalf("Failed to create pending device: %v", err)
	}

	// 2. Fetch and assert pending status
	fetched, err := repo.GetByIdentifier("carina-test-01")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}
	if fetched == nil {
		t.Fatal("Expected device to be found, got nil")
	}
	if fetched.Status != "PENDING" {
		t.Errorf("Expected status PENDING, got %s", fetched.Status)
	}
	if fetched.ApiKey != "nox_live_test_token_12345" {
		t.Errorf("Expected token nox_live_test_token_12345, got %s", fetched.ApiKey)
	}

	// 3. ValidateToken should fail while still PENDING
	valid, err := repo.ValidateToken("carina-test-01", "nox_live_test_token_12345")
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if valid {
		t.Error("Expected ValidateToken to be false for PENDING device")
	}

	// 4. Try activating with WRONG token
	activated, err := repo.ActivateDevice("carina-test-01", "wrong_token")
	if err != nil {
		t.Fatalf("ActivateDevice error: %v", err)
	}
	if activated {
		t.Error("Expected ActivateDevice to fail with wrong token")
	}

	// 5. Activate with CORRECT token
	activated, err = repo.ActivateDevice("carina-test-01", "nox_live_test_token_12345")
	if err != nil {
		t.Fatalf("ActivateDevice error: %v", err)
	}
	if !activated {
		t.Error("Expected ActivateDevice to succeed with correct token")
	}

	// 6. Assert status is now ACTIVE
	activeDev, err := repo.GetByIdentifier("carina-test-01")
	if err != nil || activeDev == nil {
		t.Fatalf("Failed to get activated device: %v", err)
	}
	if activeDev.Status != "ACTIVE" {
		t.Errorf("Expected status ACTIVE, got %s", activeDev.Status)
	}

	// 7. ValidateToken should now succeed
	valid, err = repo.ValidateToken("carina-test-01", "nox_live_test_token_12345")
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !valid {
		t.Error("Expected ValidateToken to be true for ACTIVE device with correct token")
	}

	// 8. Test CleanupExpiredPending
	past := time.Now().Add(-10 * time.Minute)
	expiredDev := &domain.Device{
		Identifier: "expired-node",
		Name:       "Expired Node",
		ApiKey:     "nox_live_expired",
		Status:     "PENDING",
		ExpiresAt:  &past,
		Enabled:    true,
	}
	if err := repo.CreateOrUpdatePending(expiredDev); err != nil {
		t.Fatalf("Failed to create expired pending device: %v", err)
	}

	if err := repo.CleanupExpiredPending(); err != nil {
		t.Fatalf("CleanupExpiredPending error: %v", err)
	}

	cleaned, err := repo.GetByIdentifier("expired-node")
	if err != nil {
		t.Fatalf("GetByIdentifier error: %v", err)
	}
	if cleaned != nil {
		t.Error("Expected expired pending device to be cleaned up, but it still exists")
	}
}

