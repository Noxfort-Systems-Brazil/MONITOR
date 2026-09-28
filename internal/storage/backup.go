// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: internal/storage/backup.go
// Author: Gabriel Moraes
// Date: 2026-09-28
// Description: Automated, non-blocking database backup engine with rotation and integrity preservation.

package storage

import (
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
