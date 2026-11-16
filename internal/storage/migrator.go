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
// File: internal/storage/migrator.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"database/sql"
	"fmt"
	"log"
)

// MigrateData atomically copies devices (including API keys, status, and expiration),
// contacts, users, settings, sessions, telemetry, and audit logs from sourceDB to targetDB.
func MigrateData(sourceDB *sql.DB, sourceDriver string, targetDB *sql.DB, targetDriver string) error {
	if sourceDB == nil || targetDB == nil {
		return fmt.Errorf("source and target databases must be initialized")
	}

	log.Printf("[MIGRATOR] Starting transactional data synchronization from %s to %s...", sourceDriver, targetDriver)

	tx, err := targetDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin target transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := migrateDevices(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateContacts(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateUsers(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateSessions(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateSettings(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateTelemetry(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateAuditLogs(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateAlertLogs(sourceDB, tx, targetDriver); err != nil {
		return err
	}
	if err := migrateTransitions(sourceDB, tx, targetDriver); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrated data: %w", err)
	}

	log.Printf("[MIGRATOR] Data synchronization completed successfully.")
	return nil
}

func migrateDevices(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query(`
		SELECT name, identifier, COALESCE(api_key, ''), COALESCE(status, 'ACTIVE'), expires_at, last_seen, enabled, created_at
		FROM devices
	`)
	if err != nil {
		return fmt.Errorf("failed querying source devices: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(`
		INSERT INTO devices (name, identifier, api_key, status, expires_at, last_seen, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (identifier) DO NOTHING;
	`, targetDriver)

	for rows.Next() {
		var name, identifier, apiKey, status string
		var expiresAt, lastSeen, createdAt sql.NullTime
		var enabled bool
		if err := rows.Scan(&name, &identifier, &apiKey, &status, &expiresAt, &lastSeen, &enabled, &createdAt); err != nil {
			return fmt.Errorf("failed scanning device during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, name, identifier, apiKey, status, expiresAt, lastSeen, enabled, createdAt); err != nil {
			return fmt.Errorf("failed inserting migrated device %q: %w", identifier, err)
		}
	}
	return rows.Err()
}

func migrateContacts(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT name, email, COALESCE(phone, ''), COALESCE(role, ''), notify_critical, enabled, COALESCE(telegram_chat_id, '') FROM contacts")
	if err != nil {
		return fmt.Errorf("failed querying source contacts: %w", err)
	}
	defer rows.Close()

	checkQuery := AdaptQuery("SELECT EXISTS(SELECT 1 FROM contacts WHERE email = ?);", targetDriver)
	insertQuery := AdaptQuery(
		"INSERT INTO contacts (name, email, phone, role, notify_critical, enabled, telegram_chat_id) VALUES (?, ?, ?, ?, ?, ?, ?);",
		targetDriver,
	)

	for rows.Next() {
		var name, email, phone, role, tg string
		var crit, en bool
		if err := rows.Scan(&name, &email, &phone, &role, &crit, &en, &tg); err != nil {
			return fmt.Errorf("failed scanning contact during migration: %w", err)
		}
		var exists bool
		if err := tx.QueryRow(checkQuery, email).Scan(&exists); err != nil {
			return fmt.Errorf("failed checking target contact duplication: %w", err)
		}
		if !exists {
			if _, err := tx.Exec(insertQuery, name, email, phone, role, crit, en, tg); err != nil {
				return fmt.Errorf("failed inserting migrated contact %q: %w", email, err)
			}
		}
	}
	return rows.Err()
}

func migrateUsers(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT username, password_hash, role, created_at FROM users")
	if err != nil {
		return fmt.Errorf("failed querying source users: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?) ON CONFLICT (username) DO NOTHING;",
		targetDriver,
	)

	for rows.Next() {
		var u, p, r string
		var t sql.NullTime
		if err := rows.Scan(&u, &p, &r, &t); err != nil {
			return fmt.Errorf("failed scanning user during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, u, p, r, t); err != nil {
			return fmt.Errorf("failed inserting migrated user %q: %w", u, err)
		}
	}
	return rows.Err()
}

func migrateSessions(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT token_hash, username, role, expires_at, created_at FROM sessions")
	if err != nil {
		return fmt.Errorf("failed querying source sessions: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO sessions (token_hash, username, role, expires_at, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT (token_hash) DO NOTHING;",
		targetDriver,
	)

	for rows.Next() {
		var hash, user, role string
		var exp, created sql.NullTime
		if err := rows.Scan(&hash, &user, &role, &exp, &created); err != nil {
			return fmt.Errorf("failed scanning session during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, hash, user, role, exp, created); err != nil {
			return fmt.Errorf("failed inserting migrated session: %w", err)
		}
	}
	return rows.Err()
}

func migrateSettings(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	var s struct {
		smtpHost, smtpUser, smtpPass, smtpFrom, adminEmail, mqtt, tg, ngTok, ngDom, duckTok, duckDom string
		smtpPort                                                                                       int
		enabled, ngEn, duckEn                                                                          bool
	}
	row := sourceDB.QueryRow(`
		SELECT COALESCE(smtp_host, ''), smtp_port, COALESCE(smtp_user, ''), COALESCE(smtp_pass, ''), COALESCE(smtp_from, ''), COALESCE(admin_email, ''),
		       COALESCE(mqtt_address, 'tcp://127.0.0.1:1883'), enabled, COALESCE(telegram_bot_token, ''),
		       COALESCE(ngrok_auth_token, ''), COALESCE(ngrok_domain, ''), COALESCE(ngrok_enabled, 0),
		       COALESCE(duckdns_token, ''), COALESCE(duckdns_domain, ''), COALESCE(duckdns_enabled, 0)
		FROM settings WHERE id = 1
	`)
	err := row.Scan(
		&s.smtpHost, &s.smtpPort, &s.smtpUser, &s.smtpPass, &s.smtpFrom, &s.adminEmail,
		&s.mqtt, &s.enabled, &s.tg, &s.ngTok, &s.ngDom, &s.ngEn,
		&s.duckTok, &s.duckDom, &s.duckEn,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed reading source settings: %w", err)
	}

	updateQuery := AdaptQuery(`
		UPDATE settings SET
			smtp_host = ?, smtp_port = ?, smtp_user = ?, smtp_pass = ?,
			smtp_from = ?, admin_email = ?, mqtt_address = ?, enabled = ?,
			telegram_bot_token = ?, ngrok_auth_token = ?, ngrok_domain = ?, ngrok_enabled = ?,
			duckdns_token = ?, duckdns_domain = ?, duckdns_enabled = ?
		WHERE id = 1;
	`, targetDriver)

	if _, err := tx.Exec(updateQuery,
		s.smtpHost, s.smtpPort, s.smtpUser, s.smtpPass,
		s.smtpFrom, s.adminEmail, s.mqtt, s.enabled,
		s.tg, s.ngTok, s.ngDom, s.ngEn,
		s.duckTok, s.duckDom, s.duckEn,
	); err != nil {
		return fmt.Errorf("failed updating target settings: %w", err)
	}
	return nil
}

func migrateTelemetry(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT COALESCE(category, ''), device_id, origin, level, COALESCE(message, ''), occurred_at FROM telemetry ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed querying source telemetry: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO telemetry (category, device_id, origin, level, message, occurred_at) VALUES (?, ?, ?, ?, ?, ?);",
		targetDriver,
	)
	for rows.Next() {
		var cat, devID, orig, lvl, msg string
		var occAt sql.NullTime
		if err := rows.Scan(&cat, &devID, &orig, &lvl, &msg, &occAt); err != nil {
			return fmt.Errorf("failed scanning telemetry during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, cat, devID, orig, lvl, msg, occAt); err != nil {
			return fmt.Errorf("failed inserting migrated telemetry: %w", err)
		}
	}
	return rows.Err()
}

func migrateAuditLogs(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT created_at, username, action, COALESCE(details, ''), COALESCE(ip_address, '') FROM security_audit_logs ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed querying source security_audit_logs: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO security_audit_logs (created_at, username, action, details, ip_address) VALUES (?, ?, ?, ?, ?);",
		targetDriver,
	)
	for rows.Next() {
		var createdAt sql.NullTime
		var user, action, details, ip string
		if err := rows.Scan(&createdAt, &user, &action, &details, &ip); err != nil {
			return fmt.Errorf("failed scanning security audit log during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, createdAt, user, action, details, ip); err != nil {
			return fmt.Errorf("failed inserting migrated security audit log: %w", err)
		}
	}
	return rows.Err()
}

func migrateAlertLogs(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT telemetry_id, channel, recipient, COALESCE(role, ''), status, COALESCE(error_reason, ''), dispatched_at FROM alert_dispatch_logs ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed querying source alert_dispatch_logs: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO alert_dispatch_logs (telemetry_id, channel, recipient, role, status, error_reason, dispatched_at) VALUES (?, ?, ?, ?, ?, ?, ?);",
		targetDriver,
	)
	for rows.Next() {
		var telID sql.NullInt64
		var ch, rec, role, st, errReas string
		var dispAt sql.NullTime
		if err := rows.Scan(&telID, &ch, &rec, &role, &st, &errReas, &dispAt); err != nil {
			return fmt.Errorf("failed scanning alert dispatch log during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, telID, ch, rec, role, st, errReas, dispAt); err != nil {
			return fmt.Errorf("failed inserting migrated alert dispatch log: %w", err)
		}
	}
	return rows.Err()
}

func migrateTransitions(sourceDB *sql.DB, tx *sql.Tx, targetDriver string) error {
	rows, err := sourceDB.Query("SELECT device_identifier, COALESCE(previous_state, ''), new_state, duration_offline_sec, transition_at FROM device_state_transitions ORDER BY id ASC")
	if err != nil {
		return fmt.Errorf("failed querying source device_state_transitions: %w", err)
	}
	defer rows.Close()

	insertQuery := AdaptQuery(
		"INSERT INTO device_state_transitions (device_identifier, previous_state, new_state, duration_offline_sec, transition_at) VALUES (?, ?, ?, ?, ?);",
		targetDriver,
	)
	for rows.Next() {
		var devID, prevState, newState string
		var dur int64
		var transAt sql.NullTime
		if err := rows.Scan(&devID, &prevState, &newState, &dur, &transAt); err != nil {
			return fmt.Errorf("failed scanning transition during migration: %w", err)
		}
		if _, err := tx.Exec(insertQuery, devID, prevState, newState, dur, transAt); err != nil {
			return fmt.Errorf("failed inserting migrated transition: %w", err)
		}
	}
	return rows.Err()
}
