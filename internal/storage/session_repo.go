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
// File: internal/storage/session_repo.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

// DBSessionStore persists authenticated user sessions in SQLite or PostgreSQL,
// storing only SHA-256 token hashes so database snapshots never leak active tokens.
type DBSessionStore struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
	ttl    time.Duration
}

// NewDBSessionStore creates a database-backed session store implementing security.SessionStore and ReloadableRepository.
func NewDBSessionStore(db *sql.DB, driver string, ttl time.Duration) *DBSessionStore {
	if driver == "" {
		driver = "sqlite"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &DBSessionStore{
		db:     db,
		driver: driver,
		ttl:    ttl,
	}
}

// SetDB updates the database connection and SQL dialect dynamically on DBManager hot-reload.
func (s *DBSessionStore) SetDB(db *sql.DB, driver string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
	if driver != "" {
		s.driver = driver
	}
}

func (s *DBSessionStore) getDB() (*sql.DB, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db, s.driver
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateSession generates a cryptographically random session token and persists its SHA-256 hash.
func (s *DBSessionStore) CreateSession(username, role string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[SESSION] Failed to generate random token: %v", err)
		return ""
	}
	token := hex.EncodeToString(b)
	tokenHash := hashSessionToken(token)
	now := time.Now()
	expiresAt := now.Add(s.ttl)

	db, driver := s.getDB()
	if db == nil {
		return ""
	}

	query := AdaptQuery(`
		INSERT INTO sessions (token_hash, username, role, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?);
	`, driver)

	if _, err := db.Exec(query, tokenHash, username, role, expiresAt, now); err != nil {
		log.Printf("[SESSION] Failed to persist session for '%s': %v", username, err)
		return ""
	}

	return token
}

// ValidateSession verifies a raw token by hashing it and validating its non-expired row in storage.
func (s *DBSessionStore) ValidateSession(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}

	db, driver := s.getDB()
	if db == nil {
		return "", "", false
	}

	tokenHash := hashSessionToken(token)
	query := AdaptQuery(`
		SELECT username, role, expires_at
		FROM sessions
		WHERE token_hash = ?
		LIMIT 1;
	`, driver)

	var username, role string
	var expiresAt time.Time
	err := db.QueryRow(query, tokenHash).Scan(&username, &role, &expiresAt)
	if err != nil {
		return "", "", false
	}

	if time.Now().After(expiresAt) {
		s.RevokeSession(token)
		return "", "", false
	}

	return username, role, true
}

// RevokeSession removes an active session by raw token.
func (s *DBSessionStore) RevokeSession(token string) {
	if token == "" {
		return
	}
	db, driver := s.getDB()
	if db == nil {
		return
	}
	query := AdaptQuery(`DELETE FROM sessions WHERE token_hash = ?;`, driver)
	_, _ = db.Exec(query, hashSessionToken(token))
}

// RevokeUserSessions invalidates all active sessions belonging to the specified username.
func (s *DBSessionStore) RevokeUserSessions(username string) {
	if username == "" {
		return
	}
	db, driver := s.getDB()
	if db == nil {
		return
	}
	query := AdaptQuery(`DELETE FROM sessions WHERE username = ?;`, driver)
	_, _ = db.Exec(query, username)
}

// CleanupExpired deletes all expired sessions from storage.
func (s *DBSessionStore) CleanupExpired() error {
	db, driver := s.getDB()
	if db == nil {
		return nil
	}
	query := AdaptQuery(`DELETE FROM sessions WHERE expires_at < ?;`, driver)
	_, err := db.Exec(query, time.Now())
	return err
}

// StartCleanup periodically purges expired sessions until ctx is cancelled.
func (s *DBSessionStore) StartCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	go func() {
		_ = s.CleanupExpired()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.CleanupExpired(); err != nil {
					log.Printf("[SESSION] Expired cleanup warning: %v", err)
				}
			}
		}
	}()
}
