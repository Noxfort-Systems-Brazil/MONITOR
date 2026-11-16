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
// File: internal/storage/retention_test.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestRetentionManager_PruneOlderThan(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	dbm := NewDBManager(db, "sqlite", domain.DatabaseConfig{Type: "sqlite"})
	telRepo := NewTelemetryRepository(db)
	auditRepo := NewAuditRepository(db, "sqlite")

	now := time.Now()
	oldTime := now.Add(-120 * 24 * time.Hour)

	if err := telRepo.SaveEvent("dev-1", &domain.IncomingEvent{
		Category: domain.CategoryHardware, Origin: "dev-1", Level: domain.LevelCritical,
		Message: "Old incident", OccurredAt: oldTime,
	}); err != nil {
		t.Fatalf("SaveEvent old failed: %v", err)
	}
	if err := telRepo.SaveEvent("dev-1", &domain.IncomingEvent{
		Category: domain.CategoryHardware, Origin: "dev-1", Level: domain.LevelWarning,
		Message: "Recent incident", OccurredAt: now.Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("SaveEvent recent failed: %v", err)
	}

	_ = auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{Username: "u", Action: "OLD", CreatedAt: oldTime})
	_ = auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{Username: "u", Action: "NEW", CreatedAt: now})

	rm := NewRetentionManager(dbm, 90)
	deleted, err := rm.PruneOlderThan(now.Add(-90 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("PruneOlderThan failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 expired rows pruned (1 telemetry + 1 audit), got %d", deleted)
	}

	remainingIncidents, err := telRepo.GetRecentIncidents(10)
	if err != nil || len(remainingIncidents) != 1 || remainingIncidents[0].Message != "Recent incident" {
		t.Fatalf("Expected only recent incident preserved, got %+v (err=%v)", remainingIncidents, err)
	}
	remainingAudit, err := auditRepo.GetRecentSecurityAuditLogs(10)
	if err != nil || len(remainingAudit) != 1 || remainingAudit[0].Action != "NEW" {
		t.Fatalf("Expected only recent audit log preserved, got %+v (err=%v)", remainingAudit, err)
	}
}
