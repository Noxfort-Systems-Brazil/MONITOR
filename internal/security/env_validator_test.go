// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

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
	os.Setenv("MONITOR_ADMIN_USER", "admin")
	os.Setenv("MONITOR_ADMIN_PASSWORD", "admin")
	defer os.Unsetenv("MONITOR_ADMIN_USER")
	defer os.Unsetenv("MONITOR_ADMIN_PASSWORD")

	reportAdmin := ValidateEnvironmentSecurity(envFile)
	if !reportAdmin.DefaultAdminAlert {
		t.Errorf("Expected DefaultAdminAlert to be true for default admin/admin")
	}
}
