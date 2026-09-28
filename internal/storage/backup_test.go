// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
	_ "modernc.org/sqlite"
)

func TestBackupManager_SQLiteBackupAndRotation(t *testing.T) {
	tempDir := t.TempDir()
	sourceDBPath := filepath.Join(tempDir, "source.db")
	backupDir := filepath.Join(tempDir, "backups")

	// 1. Create a mock SQLite database and insert dummy records
	db, err := sql.Open("sqlite", sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to open temp db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE test_data (id INTEGER PRIMARY KEY, note TEXT);"); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO test_data (note) VALUES ('industrial telemetry event');"); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	cfg := domain.DatabaseConfig{Type: "sqlite", FilePath: sourceDBPath}
	dbm := NewDBManager(db, "sqlite", cfg)

	// MaxKeep = 2 to test rotation
	bm := NewBackupManager(dbm, backupDir, 2)

	// 2. Perform first backup
	backup1, err := bm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup 1 failed: %v", err)
	}
	if backup1.SizeBytes <= 0 {
		t.Errorf("Expected backup size > 0, got: %d", backup1.SizeBytes)
	}

	// Verify that the backup file is a valid SQLite DB and contains our table
	backupDB, err := sql.Open("sqlite", backup1.FilePath)
	if err != nil {
		t.Fatalf("Failed to open backup db: %v", err)
	}
	var count int
	if err := backupDB.QueryRow("SELECT COUNT(*) FROM test_data;").Scan(&count); err != nil {
		backupDB.Close()
		t.Fatalf("Failed to read from backup db: %v", err)
	}
	backupDB.Close()
	if count != 1 {
		t.Errorf("Expected 1 record in backup db, got: %d", count)
	}

	// 3. Perform second and third backups to trigger rotation (maxKeep = 2)
	time.Sleep(1100 * time.Millisecond) // Ensure unique timestamp
	_, err = bm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup 2 failed: %v", err)
	}

	time.Sleep(1100 * time.Millisecond)
	_, err = bm.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup 3 failed: %v", err)
	}

	// 4. Verify rotation: only 2 backups should remain
	list, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 backups after rotation, found %d", len(list))
	}

	// Verify the oldest backup was removed
	if _, err := os.Stat(backup1.FilePath); !os.IsNotExist(err) {
		t.Errorf("Expected oldest backup '%s' to be deleted by rotation", backup1.FilePath)
	}
}
