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
// File: internal/security/lockout_test.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestLoginAttemptTracker_And_SecurityManagerLockout(t *testing.T) {
	now := time.Now()
	tracker := NewLoginAttemptTracker(3, 10*time.Minute, 2*time.Minute)
	tracker.nowFunc = func() time.Time { return now }

	// 1. Two failed attempts increment counter without locking
	r1 := tracker.RecordFailure("admin", "10.0.0.5")
	if r1.FailedAttempts != 1 || r1.IsLockdown {
		t.Fatalf("unexpected result on failure 1: %+v", r1)
	}
	r2 := tracker.RecordFailure("admin", "10.0.0.5")
	if r2.FailedAttempts != 2 || r2.IsLockdown {
		t.Fatalf("unexpected result on failure 2: %+v", r2)
	}

	// Different IP should remain isolated
	if _, locked := tracker.Check("admin", "10.0.0.99"); locked {
		t.Fatal("expected lockout tracking to stay scoped per client IP")
	}

	// 2. Third attempt engages lockout
	r3 := tracker.RecordFailure("admin", "10.0.0.5")
	if !r3.IsLockdown || r3.RetryAfterSeconds <= 0 {
		t.Fatalf("expected lockdown on attempt 3, got %+v", r3)
	}
	if _, locked := tracker.Check("admin", "10.0.0.5"); !locked {
		t.Fatal("expected Check to report locked=true during cooldown")
	}

	// 3. Advancing past cooldown unlocks automatically
	now = now.Add(3 * time.Minute)
	if _, locked := tracker.Check("admin", "10.0.0.5"); locked {
		t.Fatal("expected automatic unlock after cooldown duration")
	}

	// 4. End-to-end with SecurityManager (password change + unlock + role change)
	repo := newMockUserRepo()
	sm := NewSecurityManager(repo)
	sm.tracker = NewLoginAttemptTracker(2, 10*time.Minute, 5*time.Minute)
	if _, err := sm.RegisterWithRole("admin_root", "AdminRoot@2026!", domain.RoleAdmin); err != nil {
		t.Fatalf("register admin failed: %v", err)
	}
	if _, err := sm.RegisterWithRole("op_user", "OpUser@2026!", domain.RoleOperator); err != nil {
		t.Fatalf("register operator failed: %v", err)
	}

	_, attempt1, err := sm.AuthenticateWithAttempt("op_user", "wrong1", "192.168.1.50")
	if err == nil || attempt1.FailedAttempts != 1 || attempt1.IsLockdown {
		t.Fatalf("expected failure count 1 without lockdown, got attempt=%+v err=%v", attempt1, err)
	}
	_, attempt2, err := sm.AuthenticateWithAttempt("op_user", "wrong2", "192.168.1.50")
	if err == nil || !attempt2.IsLockdown {
		t.Fatalf("expected lockdown on 2nd failure, got attempt=%+v err=%v", attempt2, err)
	}

	// Even correct password is blocked while locked out
	if _, _, err := sm.AuthenticateWithAttempt("op_user", "OpUser@2026!", "192.168.1.50"); err == nil {
		t.Fatal("expected login attempt during lockdown to be rejected")
	}

	// Admin unlocks op_user and updates password (revoking active session)
	if err := sm.UnlockAccount("admin_root", "op_user", "127.0.0.1"); err != nil {
		t.Fatalf("UnlockAccount failed: %v", err)
	}
	sessTok := sm.CreateSession("op_user", domain.RoleOperator)
	if err := sm.ChangePassword("op_user", "op_user", "OpUser@2026!", "NewOpPass@2026!", "127.0.0.1"); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	if _, _, valid := sm.ValidateSession(sessTok); valid {
		t.Fatal("expected existing session to be revoked after ChangePassword")
	}
	if _, _, err := sm.AuthenticateWithAttempt("op_user", "NewOpPass@2026!", "192.168.1.50"); err != nil {
		t.Fatalf("expected authentication with new password after unlock to succeed, got %v", err)
	}
	if err := sm.UpdateUserRole("admin_root", "op_user", domain.RoleAdmin, "127.0.0.1"); err != nil {
		t.Fatalf("UpdateUserRole promotion failed: %v", err)
	}
	if err := sm.UpdateUserRole("admin_root", "op_user", domain.RoleOperator, "127.0.0.1"); err != nil {
		t.Fatalf("UpdateUserRole demotion failed: %v", err)
	}
	if err := sm.UpdateUserRole("admin_root", "admin_root", domain.RoleOperator, "127.0.0.1"); err == nil {
		t.Fatal("expected error when attempting to demote the last ADMIN")
	}
}
