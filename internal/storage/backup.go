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
// File: internal/storage/backup.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// BackupInfo aliases domain.BackupMetadata for package storage consumers.
type BackupInfo = domain.BackupMetadata

// BackupManager orchestrates hot database backups and file retention.
type BackupManager struct {
	mu        sync.Mutex
	dbManager *DBManager
	backupDir string
	maxKeep   int
}

// NewBackupManager creates an instance of BackupManager.
func NewBackupManager(dbm *DBManager, backupDir string, maxKeep int) *BackupManager {
	if backupDir == "" {
		backupDir = "backups"
	}
	if maxKeep <= 0 {
		maxKeep = 7
	}
	return &BackupManager{
		dbManager: dbm,
		backupDir: backupDir,
		maxKeep:   maxKeep,
	}
}

// CreateBackup generates an online, consistent backup of the active database.
func (b *BackupManager) CreateBackup() (BackupInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := os.MkdirAll(b.backupDir, 0700); err != nil {
		return BackupInfo{}, fmt.Errorf("failed to create backup directory '%s': %w", b.backupDir, err)
	}

	driver := "sqlite"
	var db *sql.DB
	var cfg domain.DatabaseConfig

	if b.dbManager != nil {
		driver = b.dbManager.GetDriver()
		db = b.dbManager.GetDB()
		cfg = b.dbManager.GetConfig()
	}

	timestamp := time.Now().Format("20060102_150405")
	var backupFile string
	var err error

	if driver == "postgres" {
		backupFile = filepath.Join(b.backupDir, fmt.Sprintf("monitor_postgres_%s.sql", timestamp))
		err = b.backupPostgres(cfg, backupFile)
	} else {
		backupFile = filepath.Join(b.backupDir, fmt.Sprintf("monitor_sqlite_%s.db", timestamp))
		err = b.backupSQLite(db, backupFile)
	}

	if err != nil {
		return BackupInfo{}, err
	}

	info, statErr := os.Stat(backupFile)
	if statErr != nil {
		return BackupInfo{}, fmt.Errorf("failed to verify backup file: %w", statErr)
	}

	// Rotate older backups
	_ = b.rotateBackups()

	result := BackupInfo{
		Filename:  filepath.Base(backupFile),
		FilePath:  backupFile,
		SizeBytes: info.Size(),
		Driver:    driver,
		CreatedAt: time.Now(),
	}

	log.Printf("[BACKUP] Successfully created %s backup: %s (%d bytes)", driver, result.Filename, result.SizeBytes)
	return result, nil
}

// backupSQLite performs an atomic snapshot using VACUUM INTO.
func (b *BackupManager) backupSQLite(db *sql.DB, targetPath string) error {
	if db == nil {
		return fmt.Errorf("sqlite database is not initialized")
	}

	// Sanitize path for SQL literal string escaping
	safePath := strings.ReplaceAll(targetPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s';", safePath)

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("sqlite VACUUM INTO failed: %w", err)
	}

	return nil
}

// backupPostgres invokes pg_dump if available.
func (b *BackupManager) backupPostgres(cfg domain.DatabaseConfig, targetPath string) error {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("pg_dump is not installed or not in PATH: %w", err)
	}

	args := []string{
		"-h", cfg.Host,
		"-p", fmt.Sprintf("%d", cfg.Port),
		"-U", cfg.User,
		"-d", cfg.DBName,
		"-F", "c", // Custom compressed format
		"-f", targetPath,
	}

	cmd := exec.Command("pg_dump", args...)
	if cfg.Password != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump execution failed: %v, output: %s", err, string(out))
	}

	return nil
}

// rotateBackups removes older backups exceeding the retention threshold (maxKeep).
func (b *BackupManager) rotateBackups() error {
	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		return err
	}

	type fileWithTime struct {
		path    string
		modTime time.Time
	}

	var files []fileWithTime
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "monitor_") && (strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sql")) {
			info, err := entry.Info()
			if err == nil {
				files = append(files, fileWithTime{
					path:    filepath.Join(b.backupDir, name),
					modTime: info.ModTime(),
				})
			}
		}
	}

	// Sort newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	if len(files) > b.maxKeep {
		for _, oldFile := range files[b.maxKeep:] {
			_ = os.Remove(oldFile.path)
			log.Printf("[BACKUP] Rotated old backup: %s", oldFile.path)
		}
	}

	return nil
}

// ListBackups returns all existing backups sorted by modification time descending.
func (b *BackupManager) ListBackups() ([]BackupInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "monitor_") && (strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sql")) {
			info, err := entry.Info()
			if err == nil {
				driver := "sqlite"
				if strings.HasSuffix(name, ".sql") {
					driver = "postgres"
				}
				backups = append(backups, BackupInfo{
					Filename:  name,
					FilePath:  filepath.Join(b.backupDir, name),
					SizeBytes: info.Size(),
					Driver:    driver,
					CreatedAt: info.ModTime(),
				})
			}
		}
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// ResolveBackupPath validates filename to prevent path traversal and returns its safe absolute path inside backupDir.
func (b *BackupManager) ResolveBackupPath(filename string) (string, error) {
	clean := filepath.Base(strings.TrimSpace(filename))
	if clean == "" || clean == "." || clean == ".." || clean != filename || strings.ContainsAny(clean, `/\`) {
		return "", fmt.Errorf("invalid backup filename")
	}
	if !strings.HasPrefix(clean, "monitor_") || (!strings.HasSuffix(clean, ".db") && !strings.HasSuffix(clean, ".sql")) {
		return "", fmt.Errorf("unsupported backup filename")
	}

	fullPath := filepath.Join(b.backupDir, clean)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("backup file not found: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("invalid backup target")
	}
	return fullPath, nil
}

// RestoreBackup validates and restores a snapshot into the currently configured database engine.
func (b *BackupManager) RestoreBackup(filename string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	fullPath, err := b.ResolveBackupPath(filename)
	if err != nil {
		return err
	}
	if b.dbManager == nil {
		return fmt.Errorf("db manager not initialized")
	}

	if strings.HasSuffix(fullPath, ".sql") {
		cfg := b.dbManager.GetConfig()
		if b.dbManager.GetDriver() != "postgres" {
			return fmt.Errorf("connect to PostgreSQL first before restoring a .sql backup")
		}
		if _, err := exec.LookPath("pg_restore"); err != nil {
			return fmt.Errorf("pg_restore is not installed or not in PATH: %w", err)
		}
		args := []string{
			"--clean", "--if-exists",
			"-h", cfg.Host,
			"-p", fmt.Sprintf("%d", cfg.Port),
			"-U", cfg.User,
			"-d", cfg.DBName,
			fullPath,
		}
		cmd := exec.Command("pg_restore", args...)
		if cfg.Password != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Password))
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("pg_restore execution failed: %v, output: %s", err, string(out))
		}
		return nil
	}

	// SQLite backup integrity check before touching live database
	checkDB, err := sql.Open("sqlite", fullPath)
	if err != nil {
		return fmt.Errorf("failed opening sqlite backup: %w", err)
	}
	var quickCheck string
	err = checkDB.QueryRow("PRAGMA quick_check;").Scan(&quickCheck)
	_ = checkDB.Close()
	if err != nil || !strings.EqualFold(quickCheck, "ok") {
		return fmt.Errorf("backup failed sqlite integrity check (%q): %w", quickCheck, err)
	}

	targetPath := b.dbManager.GetConfig().FilePath
	return b.dbManager.RestoreSQLiteFile(targetPath, fullPath)
}

// StartScheduledBackups runs periodic automatic snapshots in background until ctx is cancelled.
func (b *BackupManager) StartScheduledBackups(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if info, err := b.CreateBackup(); err != nil {
					log.Printf("[BACKUP] Scheduled backup failed: %v", err)
				} else {
					log.Printf("[BACKUP] Scheduled backup saved: %s (%d bytes)", info.Filename, info.SizeBytes)
				}
			}
		}
	}()
}
