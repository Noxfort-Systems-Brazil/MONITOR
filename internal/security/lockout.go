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
// File: internal/security/lockout.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"math"
	"strings"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

const (
	// DefaultMaxFailedAttempts defines how many consecutive invalid logins trigger a cooldown lock.
	DefaultMaxFailedAttempts = 5
	// DefaultAttemptWindow defines the sliding observation window before failure counts reset.
	DefaultAttemptWindow = 15 * time.Minute
	// DefaultLockoutDuration defines how long an account+IP key stays locked after reaching max attempts.
	DefaultLockoutDuration = 5 * time.Minute
)

type loginAttemptRecord struct {
	failures    int
	firstFailed time.Time
	lockedUntil time.Time
}

// LoginAttemptTracker defends authentication against credential brute-force attacks.
type LoginAttemptTracker struct {
	mu              sync.Mutex
	attempts        map[string]*loginAttemptRecord
	maxAttempts     int
	window          time.Duration
	lockoutDuration time.Duration
	nowFunc         func() time.Time
}

// NewLoginAttemptTracker creates a brute-force tracker with production defaults.
func NewLoginAttemptTracker(maxAttempts int, window, lockoutDuration time.Duration) *LoginAttemptTracker {
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxFailedAttempts
	}
	if window <= 0 {
		window = DefaultAttemptWindow
	}
	if lockoutDuration <= 0 {
		lockoutDuration = DefaultLockoutDuration
	}
	return &LoginAttemptTracker{
		attempts:        make(map[string]*loginAttemptRecord),
		maxAttempts:     maxAttempts,
		window:          window,
		lockoutDuration: lockoutDuration,
		nowFunc:         time.Now,
	}
}

func attemptKey(username, clientIP string) string {
	u := strings.ToLower(strings.TrimSpace(username))
	ip := strings.TrimSpace(clientIP)
	if ip == "" {
		return u + "|"
	}
	return u + "|" + ip
}

// Check inspects whether the (username, clientIP) pair is currently locked out.
func (t *LoginAttemptTracker) Check(username, clientIP string) (domain.AuthAttemptResult, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.nowFunc()
	key := attemptKey(username, clientIP)
	rec, ok := t.attempts[key]
	if !ok {
		return domain.AuthAttemptResult{MaxAttempts: t.maxAttempts}, false
	}

	if !rec.lockedUntil.IsZero() && now.Before(rec.lockedUntil) {
		retrySec := int(math.Ceil(rec.lockedUntil.Sub(now).Seconds()))
		if retrySec < 1 {
			retrySec = 1
		}
		return domain.AuthAttemptResult{
			FailedAttempts:    rec.failures,
			MaxAttempts:       t.maxAttempts,
			IsLockdown:        true,
			RetryAfterSeconds: retrySec,
		}, true
	}

	if (!rec.lockedUntil.IsZero() && !now.Before(rec.lockedUntil)) || now.Sub(rec.firstFailed) > t.window {
		delete(t.attempts, key)
		return domain.AuthAttemptResult{MaxAttempts: t.maxAttempts}, false
	}

	return domain.AuthAttemptResult{
		FailedAttempts: rec.failures,
		MaxAttempts:    t.maxAttempts,
	}, false
}

// RecordFailure increments the failed login count and engages lockout when maxAttempts is reached.
func (t *LoginAttemptTracker) RecordFailure(username, clientIP string) domain.AuthAttemptResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.nowFunc()
	key := attemptKey(username, clientIP)
	rec, ok := t.attempts[key]
	if !ok || now.Sub(rec.firstFailed) > t.window || (!rec.lockedUntil.IsZero() && !now.Before(rec.lockedUntil)) {
		rec = &loginAttemptRecord{firstFailed: now}
		t.attempts[key] = rec
	}

	rec.failures++
	res := domain.AuthAttemptResult{
		FailedAttempts: rec.failures,
		MaxAttempts:    t.maxAttempts,
	}

	if rec.failures >= t.maxAttempts {
		rec.lockedUntil = now.Add(t.lockoutDuration)
		res.IsLockdown = true
		res.RetryAfterSeconds = int(t.lockoutDuration.Seconds())
	}

	return res
}

// RecordSuccess clears any accumulated failure counter for the (username, clientIP) pair.
func (t *LoginAttemptTracker) RecordSuccess(username, clientIP string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, attemptKey(username, clientIP))
}

// UnlockUser removes all lockout states associated with a username (regardless of source IP).
func (t *LoginAttemptTracker) UnlockUser(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	prefix := strings.ToLower(strings.TrimSpace(username)) + "|"
	for k := range t.attempts {
		if strings.HasPrefix(k, prefix) {
			delete(t.attempts, k)
		}
	}
}
