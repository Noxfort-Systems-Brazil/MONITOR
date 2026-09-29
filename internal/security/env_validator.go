// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/security/env_validator.go
// Author: Gabriel Moraes
// Date: 2026-09-28
// Description: Audits and enforces environment file permissions and credential hygiene on startup.

package security

import (
	"log"
	"os"
	"runtime"
	"strings"
)

// SecurityAuditReport summarizes findings during environment security validation.
type SecurityAuditReport struct {
	EnvFileChecked     bool
	PermissionsEnforced bool
	DefaultAdminAlert  bool
	MqttAuthAlert      bool
	Warnings           []string
}

// ValidateEnvironmentSecurity checks the local .env file permissions and credential risks.
func ValidateEnvironmentSecurity(envPath string) SecurityAuditReport {
	report := SecurityAuditReport{
		Warnings: make([]string, 0),
	}

	if envPath == "" {
		envPath = ".env"
	}

	// 1. Check file permissions on POSIX systems
	if info, err := os.Stat(envPath); err == nil {
		report.EnvFileChecked = true
		perm := info.Mode().Perm()

		if runtime.GOOS != "windows" && (perm&0077 != 0) {
			warn := "[SECURITY] WARNING: .env file has overly permissive permissions (" + perm.String() + "). Restricting to 0600 (owner-only access)."
			log.Println(warn)
			report.Warnings = append(report.Warnings, warn)

			if chmodErr := os.Chmod(envPath, 0600); chmodErr == nil {
				report.PermissionsEnforced = true
				log.Println("[SECURITY] .env permissions successfully hardened to 0600.")
			} else {
				log.Printf("[SECURITY] Failed to auto-restrict .env permissions: %v", chmodErr)
			}
		}
	}

	// 2. Audit administrative credentials
	adminUser := os.Getenv("MONITOR_ADMIN_USER")
	adminPass := os.Getenv("MONITOR_ADMIN_PASSWORD")

	if (adminUser == "admin" || adminUser == "") && (adminPass == "admin" || adminPass == "") {
		warn := "[SECURITY] CRITICAL WARNING: Default administrative credentials ('admin'/'admin') are active! Please set a strong MONITOR_ADMIN_PASSWORD in production."
		log.Println(warn)
		report.DefaultAdminAlert = true
		report.Warnings = append(report.Warnings, warn)
	}

	// 3. Audit MQTT credentials
	mqttUser := strings.TrimSpace(os.Getenv("MQTT_USER"))
	mqttPass := strings.TrimSpace(os.Getenv("MQTT_PASSWORD"))

	if mqttUser == "" || mqttPass == "" {
		warn := "[SECURITY] NOTICE: MQTT_USER or MQTT_PASSWORD not configured in environment. Encrypted or authenticated broker will require valid credentials."
		log.Println(warn)
		report.MqttAuthAlert = true
		report.Warnings = append(report.Warnings, warn)
	}

	return report
}
