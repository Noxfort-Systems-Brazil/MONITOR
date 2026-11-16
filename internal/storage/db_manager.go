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
// File: internal/storage/db_manager.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// ReloadableRepository defines repositories that can accept a newly switched active database.
type ReloadableRepository interface {
	SetDB(db *sql.DB, driver string)
}

// DBManager orchestrates dynamic database connections, migrations, and schema provisioning.
type DBManager struct {
	mu           sync.RWMutex
	db           *sql.DB
	driver       string
	cfg          domain.DatabaseConfig
	repositories []ReloadableRepository
}

// NewDBManager initializes the central DBManager instance.
func NewDBManager(initialDB *sql.DB, driver string, cfg domain.DatabaseConfig) *DBManager {
	return &DBManager{
		db:     initialDB,
		driver: driver,
		cfg:    cfg,
	}
}

// RegisterRepository attaches a repository to receive active DB updates on hot-reload.
func (m *DBManager) RegisterRepository(repo ReloadableRepository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.repositories = append(m.repositories, repo)
	if m.db != nil {
		repo.SetDB(m.db, m.driver)
	}
}

// GetDB returns the currently active *sql.DB.
func (m *DBManager) GetDB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// GetDriver returns the active database driver ("postgres" or "sqlite").
func (m *DBManager) GetDriver() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driver
}

// GetConfig returns a copy of the active configuration.
func (m *DBManager) GetConfig() domain.DatabaseConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// TestConnection tests connectivity against the provided configuration.
func (m *DBManager) TestConnection(cfg domain.DatabaseConfig) (domain.DatabaseStatus, error) {
	start := time.Now()
	status := domain.DatabaseStatus{
		Type:   cfg.Type,
		Host:   cfg.Host,
		Port:   cfg.Port,
		DBName: cfg.DBName,
		Schema: cfg.Schema,
		User:   cfg.User,
	}

	dialect, err := GetDialect(cfg.Type)
	if err != nil {
		status.Connected = false
		status.ErrorMessage = err.Error()
		return status, err
	}

	testDB, err := dialect.Open(cfg)
	if err != nil {
		status.Connected = false
		status.ErrorMessage = err.Error()
		return status, err
	}
	defer testDB.Close()

	status.LatencyMs = time.Since(start).Milliseconds()
	status.Connected = true
	status.ServerTime = time.Now()

	version, _ := dialect.GetVersion(testDB)
	status.Version = version

	schemaExists, _ := dialect.CheckSchemaExists(testDB, cfg.Schema)
	status.SchemaExists = schemaExists

	return status, nil
}

// Switch hot-reloads the active database connection to the new configuration.
func (m *DBManager) Switch(newCfg domain.DatabaseConfig, migrate bool) error {
	dialect, err := GetDialect(newCfg.Type)
	if err != nil {
		return fmt.Errorf("failed to resolve database dialect: %w", err)
	}

	newDB, err := dialect.Open(newCfg)
	if err != nil {
		return fmt.Errorf("failed to open new database: %w", err)
	}

	// Provision Schema and Tables via dialect
	if err := dialect.InitSchema(newDB, newCfg.Schema); err != nil {
		newDB.Close()
		return fmt.Errorf("failed to initialize schema for %s: %w", dialect.DriverName(), err)
	}

	m.mu.Lock()
	oldDB := m.db
	oldDriver := m.driver
	driver := dialect.DriverName()

	// Optional transactional data migration (abort switch if migration fails)
	if migrate && oldDB != nil {
		if err := MigrateData(oldDB, oldDriver, newDB, driver); err != nil {
			m.mu.Unlock()
			_ = newDB.Close()
			return fmt.Errorf("data migration failed: %w", err)
		}
	}

	// Update manager state
	m.db = newDB
	m.driver = driver
	m.cfg = newCfg

	// Save configuration persistently
	_ = SaveDatabaseConfig(newCfg)

	// Propagate new DB to all registered repositories
	for _, r := range m.repositories {
		r.SetDB(newDB, driver)
	}

	m.mu.Unlock()

	// Gracefully close old connection
	if oldDB != nil {
		_ = oldDB.Close()
	}

	log.Printf("[STORAGE] Successfully switched active database to %s (%s).", driver, newCfg.DBName)
	return nil
}

// RestoreSQLiteFile safely replaces the active local SQLite file with a validated backup file,
// cleaning WAL/SHM sidecars and reloading every registered repository atomically.
func (m *DBManager) RestoreSQLiteFile(targetPath, backupPath string) error {
	if targetPath == "" || targetPath == ":memory:" {
		return fmt.Errorf("cannot restore file over non-persistent sqlite database")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		_ = m.db.Close()
		m.db = nil
	}
	_ = os.Remove(targetPath + "-wal")
	_ = os.Remove(targetPath + "-shm")

	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer src.Close()

	tmpPath := targetPath + ".restoring"
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to stage restore file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed copying backup contents: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed flushing restored file: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed replacing target sqlite file: %w", err)
	}

	newCfg := m.cfg
	newCfg.Type = "sqlite"
	newCfg.FilePath = targetPath
	newDB, err := NewDatabase(targetPath)
	if err != nil {
		return fmt.Errorf("failed reopening restored sqlite database: %w", err)
	}

	m.db = newDB
	m.driver = "sqlite"
	m.cfg = newCfg
	_ = SaveDatabaseConfig(newCfg)

	for _, r := range m.repositories {
		r.SetDB(newDB, "sqlite")
	}

	log.Printf("[STORAGE] Restored SQLite database from backup '%s' to '%s'.", backupPath, targetPath)
	return nil
}

// GetStatus returns the current live health of the active database connection.
func (m *DBManager) GetStatus() domain.DatabaseStatus {
	m.mu.RLock()
	db := m.db
	driver := m.driver
	cfg := m.cfg
	m.mu.RUnlock()

	status := domain.DatabaseStatus{
		Type:   driver,
		Host:   cfg.Host,
		Port:   cfg.Port,
		DBName: cfg.DBName,
		Schema: cfg.Schema,
		User:   cfg.User,
	}

	if db == nil {
		status.Connected = false
		status.ErrorMessage = "Database not initialized"
		return status
	}

	start := time.Now()
	if err := db.Ping(); err != nil {
		status.Connected = false
		status.ErrorMessage = err.Error()
		return status
	}

	status.Connected = true
	status.LatencyMs = time.Since(start).Milliseconds()
	status.ServerTime = time.Now()

	if dialect, err := GetDialect(driver); err == nil {
		version, _ := dialect.GetVersion(db)
		status.Version = version

		schemaExists, _ := dialect.CheckSchemaExists(db, cfg.Schema)
		status.SchemaExists = schemaExists
	}

	return status
}
