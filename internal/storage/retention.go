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
// File: internal/storage/retention.go
// Author: Gabriel Moraes
// Date: October 2026

package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RetentionManager prunes aged telemetry and audit logs so continuous 24/7 operation stays bounded on disk.
type RetentionManager struct {
	dbManager     *DBManager
	retentionDays int
}

// NewRetentionManager constructs a RetentionManager with a retention horizon in days (<= 0 disables pruning).
func NewRetentionManager(dbm *DBManager, retentionDays int) *RetentionManager {
	return &RetentionManager{
		dbManager:     dbm,
		retentionDays: retentionDays,
	}
}

type retentionRule struct {
	table  string
	column string
}

var defaultRetentionRules = []retentionRule{
	{table: "telemetry", column: "occurred_at"},
	{table: "alert_dispatch_logs", column: "dispatched_at"},
	{table: "security_audit_logs", column: "created_at"},
	{table: "device_state_transitions", column: "transition_at"},
}

// PruneOlderThan deletes records older than cutoff across telemetry and audit history tables.
func (r *RetentionManager) PruneOlderThan(cutoff time.Time) (int64, error) {
	if r.dbManager == nil {
		return 0, fmt.Errorf("db manager is required")
	}
	db := r.dbManager.GetDB()
	driver := r.dbManager.GetDriver()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var totalDeleted int64
	for _, rule := range defaultRetentionRules {
		query := AdaptQuery(fmt.Sprintf(`DELETE FROM %s WHERE %s < ?;`, rule.table, rule.column), driver)
		res, err := db.Exec(query, cutoff)
		if err != nil {
			return totalDeleted, fmt.Errorf("retention prune failed on %s: %w", rule.table, err)
		}
		if n, err := res.RowsAffected(); err == nil {
			totalDeleted += n
		}
	}
	return totalDeleted, nil
}

// Start launches a periodic background pruning worker until ctx is cancelled.
func (r *RetentionManager) Start(ctx context.Context, interval time.Duration) {
	if r.retentionDays <= 0 {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	runOnce := func() {
		cutoff := time.Now().Add(-time.Duration(r.retentionDays) * 24 * time.Hour)
		deleted, err := r.PruneOlderThan(cutoff)
		if err != nil {
			log.Printf("[RETENTION] Pruning warning: %v", err)
			return
		}
		if deleted > 0 {
			log.Printf("[RETENTION] Pruned %d records older than %d days.", deleted, r.retentionDays)
		}
	}

	go func() {
		runOnce()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
}
