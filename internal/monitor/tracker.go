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
// File: internal/monitor/tracker.go
// Author: Gabriel Moraes
// Date: October 2026

package monitor

import (
	"sync"
	"time"
)

// SystemStatusTracker manages the online/offline presence state in memory to suppress alert spam.
type SystemStatusTracker struct {
	mu           sync.RWMutex
	offlineSince map[string]time.Time
}

// NewSystemStatusTracker initializes an empty tracker.
func NewSystemStatusTracker() *SystemStatusTracker {
	return &SystemStatusTracker{
		offlineSince: make(map[string]time.Time),
	}
}

// SeedOffline registers a system as offline on boot to avoid false-positive alerts on the first cycle.
func (t *SystemStatusTracker) SeedOffline(identifier string, since time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if since.IsZero() {
		since = time.Now()
	}
	t.offlineSince[identifier] = since
}

// MarkOffline registers a system as offline.
// It returns true only if the system transitioned from online to offline (first detection).
func (t *SystemStatusTracker) MarkOffline(identifier string) bool {
	return t.MarkOfflineWithTime(identifier, time.Now())
}

// MarkOfflineWithTime registers a system as offline with a specific timestamp.
func (t *SystemStatusTracker) MarkOfflineWithTime(identifier string, since time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.offlineSince[identifier]; exists {
		return false
	}

	if since.IsZero() {
		since = time.Now()
	}
	t.offlineSince[identifier] = since
	return true
}

// MarkOnline registers a system as recovered.
// It returns true only if the system transitioned from offline to online (recovery detection).
func (t *SystemStatusTracker) MarkOnline(identifier string) bool {
	recovered, _ := t.MarkOnlineWithDuration(identifier)
	return recovered
}

// MarkOnlineWithDuration registers a system as recovered and returns the elapsed offline duration.
func (t *SystemStatusTracker) MarkOnlineWithDuration(identifier string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	since, exists := t.offlineSince[identifier]
	if !exists {
		return false, 0
	}

	duration := time.Since(since)
	delete(t.offlineSince, identifier)
	return true, duration
}

// IsOffline checks if an identifier is currently considered offline.
func (t *SystemStatusTracker) IsOffline(identifier string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.offlineSince[identifier]
	return exists
}

// Reset clears all tracked presence states.
func (t *SystemStatusTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.offlineSince = make(map[string]time.Time)
}
