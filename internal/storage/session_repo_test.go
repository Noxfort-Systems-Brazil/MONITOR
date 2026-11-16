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
// File: internal/storage/session_repo_test.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestDBSessionStore_LifecycleAndHotReload(t *testing.T) {
	tempDir := t.TempDir()
	db1Path := filepath.Join(tempDir, "sessions1.db")
	db2Path := filepath.Join(tempDir, "sessions2.db")

	db1, err := NewDatabase(db1Path)
	if err != nil {
		t.Fatalf("failed to open db1: %v", err)
	}
	defer db1.Close()

	store := NewDBSessionStore(db1, "sqlite", 2*time.Hour)
	token := store.CreateSession("alice", domain.RoleOperator)
	if token == "" {
		t.Fatal("expected non-empty session token")
	}

	// Ensure raw token is NOT persisted in plaintext
	var rawCount int
	if err := db1.QueryRow("SELECT COUNT(*) FROM sessions WHERE token_hash = ?", token).Scan(&rawCount); err != nil {
		t.Fatalf("failed checking stored token hash: %v", err)
	}
	if rawCount != 0 {
		t.Fatalf("SECURITY VIOLATION: raw session token stored unhashed in database")
	}

	// Re-create store pointing to same SQLite file to verify persistence across process restarts
	restartedStore := NewDBSessionStore(db1, "sqlite", 2*time.Hour)
	user, role, ok := restartedStore.ValidateSession(token)
	if !ok || user != "alice" || role != domain.RoleOperator {
		t.Fatalf("expected persisted session for alice/OPERATOR, got user=%q role=%q ok=%v", user, role, ok)
	}

	// Revoke all sessions for alice
	restartedStore.CreateSession("bob", domain.RoleAdmin)
	restartedStore.RevokeUserSessions("alice")
	if _, _, ok := restartedStore.ValidateSession(token); ok {
		t.Fatal("expected alice session to be invalid after RevokeUserSessions")
	}

	// Hot-reload via SetDB
	db2, err := NewDatabase(db2Path)
	if err != nil {
		t.Fatalf("failed to open db2: %v", err)
	}
	defer db2.Close()

	store.SetDB(db2, "sqlite")
	bobToken := store.CreateSession("bob", domain.RoleAdmin)
	if u, r, valid := store.ValidateSession(bobToken); !valid || u != "bob" || r != domain.RoleAdmin {
		t.Fatalf("expected session in hot-reloaded db2, got user=%q role=%q valid=%v", u, r, valid)
	}

	// Expired session cleanup
	shortStore := NewDBSessionStore(db2, "sqlite", 10*time.Millisecond)
	expToken := shortStore.CreateSession("temp_user", domain.RoleOperator)
	time.Sleep(25 * time.Millisecond)
	if _, _, valid := shortStore.ValidateSession(expToken); valid {
		t.Fatal("expected expired session to be rejected")
	}
}
