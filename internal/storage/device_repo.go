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
// File: internal/storage/device_repo.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DeviceRepository manages the lifecycle of monitored systems in the database.
type DeviceRepository struct {
	mu               sync.RWMutex
	db               *sql.DB
	driver           string
	debounceMu       sync.Mutex
	lastPersisted    map[string]time.Time
	debounceInterval time.Duration
}

// NewDeviceRepository initializes the repository.
func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{
		db:               db,
		driver:           "sqlite",
		lastPersisted:    make(map[string]time.Time),
		debounceInterval: 10 * time.Second,
	}
}

// SetDebounceInterval configures the in-memory throttle window for heartbeat writes.
// Setting interval to 0 disables debouncing.
func (r *DeviceRepository) SetDebounceInterval(interval time.Duration) {
	r.debounceMu.Lock()
	defer r.debounceMu.Unlock()
	r.debounceInterval = interval
}

// SetDB updates the database connection and dialect at runtime.
func (r *DeviceRepository) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	r.db = db
	r.driver = driver
	r.mu.Unlock()

	r.debounceMu.Lock()
	r.lastPersisted = make(map[string]time.Time)
	r.debounceMu.Unlock()
}

func (r *DeviceRepository) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// GetAllDevices retrieves all registered systems for the dashboard.
// It supplements persisted data with hot in-memory timestamps from recent heartbeats.
func (r *DeviceRepository) GetAllDevices() ([]domain.Device, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT id, name, identifier, api_key, status, expires_at, last_seen, enabled FROM devices ORDER BY name ASC;`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var d domain.Device
		var apiKey sql.NullString
		var status sql.NullString
		var expiresAt sql.NullTime
		var lastSeen sql.NullTime

		if err := rows.Scan(&d.ID, &d.Name, &d.Identifier, &apiKey, &status, &expiresAt, &lastSeen, &d.Enabled); err != nil {
			return nil, err
		}

		if apiKey.Valid {
			d.ApiKey = apiKey.String
		}
		if status.Valid && status.String != "" {
			d.Status = status.String
		} else {
			d.Status = "ACTIVE"
		}
		if expiresAt.Valid {
			d.ExpiresAt = &expiresAt.Time
		}
		if lastSeen.Valid {
			d.LastSeen = lastSeen.Time
		}
		devices = append(devices, d)
	}

	// Overlay warm in-memory heartbeat timestamps for real-time dashboard accuracy
	r.debounceMu.Lock()
	for i := range devices {
		if cached, ok := r.lastPersisted[devices[i].Identifier]; ok && cached.After(devices[i].LastSeen) {
			devices[i].LastSeen = cached
		}
	}
	r.debounceMu.Unlock()

	return devices, nil
}

// GetByIdentifier retrieves a single device by its origin/identifier.
func (r *DeviceRepository) GetByIdentifier(identifier string) (*domain.Device, error) {
	db, driver := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`SELECT id, name, identifier, api_key, status, expires_at, last_seen, enabled FROM devices WHERE identifier = ? LIMIT 1;`, driver)
	row := db.QueryRow(query, identifier)

	var d domain.Device
	var apiKey sql.NullString
	var status sql.NullString
	var expiresAt sql.NullTime
	var lastSeen sql.NullTime

	if err := row.Scan(&d.ID, &d.Name, &d.Identifier, &apiKey, &status, &expiresAt, &lastSeen, &d.Enabled); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get device %s: %w", identifier, err)
	}

	if apiKey.Valid {
		d.ApiKey = apiKey.String
	}
	if status.Valid && status.String != "" {
		d.Status = status.String
	} else {
		d.Status = "ACTIVE"
	}
	if expiresAt.Valid {
		d.ExpiresAt = &expiresAt.Time
	}
	if lastSeen.Valid {
		d.LastSeen = lastSeen.Time
	}

	r.debounceMu.Lock()
	if cached, ok := r.lastPersisted[identifier]; ok && cached.After(d.LastSeen) {
		d.LastSeen = cached
	}
	r.debounceMu.Unlock()

	return &d, nil
}

// CreateOrUpdatePending stores a device in PENDING state with its secret API key and expiration TTL.
func (r *DeviceRepository) CreateOrUpdatePending(device *domain.Device) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
	INSERT INTO devices (name, identifier, api_key, status, expires_at, enabled)
	VALUES (?, ?, ?, 'PENDING', ?, 1)
	ON CONFLICT(identifier) DO UPDATE SET
		name = excluded.name,
		api_key = excluded.api_key,
		status = 'PENDING',
		expires_at = excluded.expires_at,
		enabled = 1;
	`
	if driver == "postgres" {
		query = `
		INSERT INTO devices (name, identifier, api_key, status, expires_at, enabled)
		VALUES ($1, $2, $3, 'PENDING', $4, TRUE)
		ON CONFLICT(identifier) DO UPDATE SET
			name = EXCLUDED.name,
			api_key = EXCLUDED.api_key,
			status = 'PENDING',
			expires_at = EXCLUDED.expires_at,
			enabled = TRUE;
		`
	}

	_, err := db.Exec(query, device.Name, device.Identifier, device.ApiKey, device.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to register pending device %s: %w", device.Identifier, err)
	}
	return nil
}

// ActivateDevice verifies if the device is pending, validates the token and timestamp, and promotes it to ACTIVE.
func (r *DeviceRepository) ActivateDevice(identifier string, token string) (bool, error) {
	device, err := r.GetByIdentifier(identifier)
	if err != nil {
		return false, err
	}
	if device == nil {
		return false, nil
	}

	// Verify token match
	if device.ApiKey != token {
		return false, nil
	}

	// If already active, it's valid
	if device.Status == "ACTIVE" {
		return true, nil
	}

	// If pending, check expiration TTL
	if device.Status == "PENDING" {
		if device.ExpiresAt != nil && time.Now().After(*device.ExpiresAt) {
			// TTL exceeded: clean up expired entry
			_ = r.DeleteDevice(identifier)
			return false, fmt.Errorf("pairing session expired")
		}

		db, driver := r.getDB()
		if db == nil {
			return false, fmt.Errorf("database not initialized")
		}

		now := time.Now()
		query := AdaptQuery(`UPDATE devices SET status = 'ACTIVE', expires_at = NULL, last_seen = ? WHERE identifier = ?;`, driver)
		if _, err := db.Exec(query, now, identifier); err != nil {
			return false, fmt.Errorf("failed to activate device %s: %w", identifier, err)
		}

		r.debounceMu.Lock()
		if r.lastPersisted != nil {
			r.lastPersisted[identifier] = now
		}
		r.debounceMu.Unlock()

		return true, nil
	}

	return false, nil
}

// ValidateToken verifies that a device exists, is enabled, is ACTIVE, and possesses the matching API key.
func (r *DeviceRepository) ValidateToken(identifier string, token string) (bool, error) {
	device, err := r.GetByIdentifier(identifier)
	if err != nil {
		return false, err
	}
	if device == nil || !device.Enabled {
		return false, nil
	}

	// If the device has no API key set (legacy dev environment), allow it
	if device.ApiKey == "" {
		return true, nil
	}

	if device.Status == "ACTIVE" && device.ApiKey == token {
		return true, nil
	}

	return false, nil
}

// CleanupExpiredPending deletes any devices remaining in PENDING state after their TTL.
func (r *DeviceRepository) CleanupExpiredPending() error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM devices WHERE status = 'PENDING' AND expires_at IS NOT NULL AND expires_at < ?;`, driver)
	_, err := db.Exec(query, time.Now())
	return err
}

// UpdateLastSeen updates the heartbeat for a specific system.
// AUTO-DISCOVERY: If the 'identifier' (origin) doesn't exist, it creates it.
// DEBOUNCED: If an update for the same device occurred within debounceInterval,
// the in-memory timestamp is updated immediately and database write is debounced.
func (r *DeviceRepository) UpdateLastSeen(identifier string, seenAt time.Time) error {
	if seenAt.IsZero() {
		seenAt = time.Now()
	}

	r.debounceMu.Lock()
	if r.lastPersisted == nil {
		r.lastPersisted = make(map[string]time.Time)
	}
	last, exists := r.lastPersisted[identifier]
	interval := r.debounceInterval
	// If recently persisted and within debounce window, update in-memory timestamp and skip DB disk write
	if interval > 0 && exists && seenAt.Sub(last) < interval && time.Since(last) < interval {
		r.lastPersisted[identifier] = seenAt
		r.debounceMu.Unlock()
		return nil
	}
	r.lastPersisted[identifier] = seenAt
	r.debounceMu.Unlock()

	return r.ForceUpdateLastSeen(identifier, seenAt)
}

// ForceUpdateLastSeen immediately persists the heartbeat to the database, bypassing the debounce cache.
func (r *DeviceRepository) ForceUpdateLastSeen(identifier string, seenAt time.Time) error {
	if seenAt.IsZero() {
		seenAt = time.Now()
	}

	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
	INSERT INTO devices (name, identifier, status, last_seen, enabled)
	VALUES (?, ?, 'ACTIVE', ?, 1)
	ON CONFLICT(identifier) DO UPDATE SET last_seen = excluded.last_seen, enabled = 1;
	`
	if driver == "postgres" {
		query = `
		INSERT INTO devices (name, identifier, status, last_seen, enabled)
		VALUES ($1, $2, 'ACTIVE', $3, TRUE)
		ON CONFLICT(identifier) DO UPDATE SET last_seen = EXCLUDED.last_seen, enabled = TRUE;
		`
	}

	defaultName := identifier

	_, err := db.Exec(query, defaultName, identifier, seenAt)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat for %s: %w", identifier, err)
	}

	return nil
}

// DeleteDevice removes a system permanently by its identifier and clears its debounce cache.
func (r *DeviceRepository) DeleteDevice(identifier string) error {
	r.debounceMu.Lock()
	if r.lastPersisted != nil {
		delete(r.lastPersisted, identifier)
	}
	r.debounceMu.Unlock()

	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM devices WHERE identifier = ?;`, driver)
	_, err := db.Exec(query, identifier)
	return err
}

