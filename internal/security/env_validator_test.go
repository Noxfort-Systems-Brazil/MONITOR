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
// File: internal/security/env_validator_test.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEnvironmentSecurity(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	// Create a test .env with overly permissive permissions (0666)
	if err := os.WriteFile(envFile, []byte("MONITOR_ADMIN_USER=admin\nMONITOR_ADMIN_PASSWORD=admin\n"), 0666); err != nil {
		t.Fatalf("Failed to create temp .env: %v", err)
	}

	report := ValidateEnvironmentSecurity(envFile)

	if !report.EnvFileChecked {
		t.Errorf("Expected EnvFileChecked to be true")
	}

	// Verify permission hardening
	info, err := os.Stat(envFile)
	if err != nil {
		t.Fatalf("Failed to stat temp .env: %v", err)
	}

	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		t.Errorf("Expected .env permissions to be restricted to 0600, got: %v", perm)
	}

	// Test default admin detection
	os.Setenv("MONITOR_ADMIN_USER", DefaultSuperuserUsername)
	os.Setenv("MONITOR_ADMIN_PASSWORD", DefaultSuperuserPassword)
	defer os.Unsetenv("MONITOR_ADMIN_USER")
	defer os.Unsetenv("MONITOR_ADMIN_PASSWORD")

	reportAdmin := ValidateEnvironmentSecurity(envFile)
	if !reportAdmin.DefaultAdminAlert {
		t.Errorf("Expected DefaultAdminAlert to be true for default admin credentials")
	}
}
