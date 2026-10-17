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
// File: internal/storage/migrator_test.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestMigrateData_FullCoverage(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.db")
	targetPath := filepath.Join(tempDir, "target.db")

	sourceDB, err := NewDatabase(sourcePath)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer sourceDB.Close()

	targetDB, err := NewDatabase(targetPath)
	if err != nil {
		t.Fatalf("Failed to create target DB: %v", err)
	}
	defer targetDB.Close()

	// 1. Populate source with data
	devRepo := NewDeviceRepository(sourceDB)
	_ = devRepo.UpdateLastSeen("test-origin", time.Now())

	contactRepo := NewContactRepository(sourceDB)
	_ = contactRepo.CreateContact(&domain.Contact{
		Name:           "Operator John",
		Email:          "john@example.com",
		Role:           "technician",
		NotifyCritical: true,
		Enabled:        true,
	})

	userRepo := NewUserRepository(sourceDB)
	_ = userRepo.Create(&domain.User{
		Username:     "john_tech",
		PasswordHash: "salt:hash",
		Role:         domain.RoleOperator,
		CreatedAt:    time.Now(),
	})

	telRepo := NewTelemetryRepository(sourceDB)
	_ = telRepo.SaveEvent("test-origin", &domain.IncomingEvent{
		Category:   domain.CategoryHardware,
		Origin:     "test-origin",
		Level:      domain.LevelCritical,
		Message:    "Overheating detected",
		OccurredAt: time.Now(),
	})

	auditRepo := NewAuditRepository(sourceDB, "sqlite")
	_ = auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
		Username:  "admin",
		Action:    "LOGIN_SUCCESS",
		Details:   "Test audit",
		IPAddress: "127.0.0.1",
		CreatedAt: time.Now(),
	})
	_ = auditRepo.SaveDeviceStateTransition(&domain.DeviceStateTransition{
		DeviceIdentifier:   "test-origin",
		PreviousState:      "ONLINE",
		NewState:           "OFFLINE",
		DurationOfflineSec: 120,
		TransitionAt:       time.Now(),
	})

	// 2. Perform Migration
	err = MigrateData(sourceDB, "sqlite", targetDB, "sqlite")
	if err != nil {
		t.Fatalf("MigrateData returned error: %v", err)
	}

	// 3. Verify target has the migrated data
	targetDevRepo := NewDeviceRepository(targetDB)
	devices, err := targetDevRepo.GetAllDevices()
	if err != nil || len(devices) == 0 {
		t.Errorf("Expected migrated devices in target DB")
	}

	targetTelRepo := NewTelemetryRepository(targetDB)
	incidents, err := targetTelRepo.GetRecentIncidents(10)
	if err != nil || len(incidents) == 0 {
		t.Errorf("Expected migrated telemetry incidents in target DB, got %d", len(incidents))
	}

	targetAuditRepo := NewAuditRepository(targetDB, "sqlite")
	secLogs, err := targetAuditRepo.GetRecentSecurityAuditLogs(10)
	if err != nil || len(secLogs) == 0 {
		t.Errorf("Expected migrated security audit logs in target DB, got %d", len(secLogs))
	}

	transLogs, err := targetAuditRepo.GetRecentDeviceStateTransitions(10)
	if err != nil || len(transLogs) == 0 {
		t.Errorf("Expected migrated state transitions in target DB, got %d", len(transLogs))
	}
}
