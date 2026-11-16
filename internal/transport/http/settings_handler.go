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
// File: internal/transport/http/settings_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// ConnectionTester defines the contract required to test notification channel credentials.
type ConnectionTester interface {
	TestConnection(settings *domain.Settings, to string) error
	TestTelegramConnection(botToken, chatID string) error
}

// SettingsHandler manages the configuration page and actions.
type SettingsHandler struct {
	repo           domain.SettingsRepository
	tester         ConnectionTester
	auditRepo      domain.AuditRepository
	onMQTTChanged  func(string)
}

// NewSettingsHandler creates a handler for system settings with an injected ConnectionTester.
func NewSettingsHandler(r domain.SettingsRepository, t ConnectionTester) *SettingsHandler {
	return &SettingsHandler{
		repo:   r,
		tester: t,
	}
}

// SetAuditRepository attaches an AuditRepository for recording settings updates.
func (h *SettingsHandler) SetAuditRepository(repo domain.AuditRepository) {
	h.auditRepo = repo
}

// SetOnMQTTConfigChanged registers a live reconnect callback invoked when mqtt_address changes.
func (h *SettingsHandler) SetOnMQTTConfigChanged(cb func(string)) {
	h.onMQTTChanged = cb
}

// mergeSettingsFromRequest selectively updates configuration sections without wiping unrelated fields or stored secrets.
func mergeSettingsFromRequest(r *http.Request, existing *domain.Settings) *domain.Settings {
	merged := &domain.Settings{
		ID:          1,
		SMTPPort:    587,
		MqttAddress: "tcp://127.0.0.1:1883",
		Enabled:     true,
	}
	if existing != nil {
		copyVal := *existing
		merged = &copyVal
		merged.ID = 1
	}

	section := strings.TrimSpace(r.FormValue("form_section"))

	if section == "" || section == "smtp" {
		port, err := strconv.Atoi(r.FormValue("smtp_port"))
		if err == nil && port > 0 {
			merged.SMTPPort = port
		}
		smtpUser := strings.TrimSpace(r.FormValue("smtp_user"))
		adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
		if adminEmail == "" {
			adminEmail = smtpUser
		}
		smtpPass := r.FormValue("smtp_pass")
		// Preserve stored SMTP password if blank and the SMTP username did not change
		if smtpPass == "" && existing != nil && existing.SMTPPass != "" && (smtpUser == "" || smtpUser == existing.SMTPUser) {
			smtpPass = existing.SMTPPass
		}

		merged.SMTPHost = strings.TrimSpace(r.FormValue("smtp_host"))
		merged.SMTPUser = smtpUser
		merged.SMTPPass = smtpPass
		merged.SMTPFrom = strings.TrimSpace(r.FormValue("smtp_from"))
		merged.AdminEmail = adminEmail
		merged.Enabled = true
	}

	if section == "" || section == "telegram" {
		if r.FormValue("clear_telegram") == "true" {
			merged.TelegramBotToken = ""
		} else if tok := strings.TrimSpace(r.FormValue("telegram_bot_token")); tok != "" {
			merged.TelegramBotToken = tok
		}
	}

	if section == "" || section == "mqtt" {
		if mqttAddr := strings.TrimSpace(r.FormValue("mqtt_address")); mqttAddr != "" {
			merged.MqttAddress = mqttAddr
		}
	}

	return merged
}

func maskTokenPreview(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return ""
	}
	if len(tok) <= 4 {
		return "••••"
	}
	return tok[:4] + "•••••••••"
}

// ServePage renders the settings form without exposing plaintext credentials in the HTML DOM.
func (h *SettingsHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	viewSettings := domain.Settings{}
	if settings != nil {
		viewSettings = *settings
	}
	hasSMTPPass := viewSettings.SMTPPass != ""
	hasTelegramToken := viewSettings.TelegramBotToken != ""
	telegramHint := maskTokenPreview(viewSettings.TelegramBotToken)
	// Redact plaintext secrets before HTML template rendering
	viewSettings.SMTPPass = ""
	viewSettings.TelegramBotToken = ""
	viewSettings.DuckDNSToken = ""
	viewSettings.NgrokAuthToken = ""

	data := map[string]interface{}{
		"Title":             "System Settings",
		"Settings":          &viewSettings,
		"HasSMTPPass":       hasSMTPPass,
		"HasTelegramToken":  hasTelegramToken,
		"TelegramTokenHint": telegramHint,
	}

	if err := RenderTemplateWithRequest(w, r, data,
		"web/templates/layout.html",
		"web/templates/settings.html",
		"web/templates/partials/settings_smtp.html",
		"web/templates/partials/settings_telegram.html",
	); err != nil {
		log.Printf("[SETTINGS] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}
}

// HandleSave processes the form submission to update settings.
func (h *SettingsHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	existing, _ := h.repo.GetSettings()
	settings := mergeSettingsFromRequest(r, existing)

	// Persist to database
	if err := h.repo.SaveSettings(settings); err != nil {
		log.Printf("[SETTINGS] Failed to save: %v", err)
		http.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	log.Println("[SETTINGS] Configuration updated successfully.")
	section := strings.TrimSpace(r.FormValue("form_section"))
	if section == "" {
		section = "general"
	}
	if h.onMQTTChanged != nil && (section == "mqtt" || section == "general") && settings.MqttAddress != "" {
		if existing == nil || existing.MqttAddress != settings.MqttAddress {
			h.onMQTTChanged(settings.MqttAddress)
		}
	}
	logSecurityAuditEvent(h.auditRepo, r, "SETTINGS_UPDATED", "Configurações atualizadas (seção: "+section+")")
	Redirect(w, r, "/settings", http.StatusSeeOther)
}

// HandleTest sends a test email to the SMTP User themselves to validate the connection.
func (h *SettingsHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings for test: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if settings.SMTPUser == "" {
		http.Error(w, "No account configured. Please Connect Account first.", http.StatusBadRequest)
		return
	}

	targetEmail := settings.SMTPUser
	log.Printf("[SETTINGS] Triggering self-test email to %s via %s:%d...",
		targetEmail, settings.SMTPHost, settings.SMTPPort)

	if h.tester != nil {
		if err := h.tester.TestConnection(settings, targetEmail); err != nil {
			log.Printf("[SETTINGS] Test failed: %v", err)
			http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, _ = w.Write([]byte("Test email sent to your inbox!"))
}

// HandleTestTelegram sends a test Telegram message to validate the bot token and chat ID.
func (h *SettingsHandler) HandleTestTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	settings, err := h.repo.GetSettings()
	if err != nil {
		log.Printf("[SETTINGS] Failed to load settings for Telegram test: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	if settings.TelegramBotToken == "" {
		http.Error(w, "No Telegram bot configured. Save a bot token first.", http.StatusBadRequest)
		return
	}

	chatID := strings.TrimSpace(r.FormValue("chat_id"))
	if chatID == "" {
		http.Error(w, "Please provide a chat_id to send the test message.", http.StatusBadRequest)
		return
	}

	log.Printf("[SETTINGS] Sending Telegram test to chat_id %s...", chatID)

	if h.tester != nil {
		if err := h.tester.TestTelegramConnection(settings.TelegramBotToken, chatID); err != nil {
			log.Printf("[SETTINGS] Telegram test failed: %v", err)
			http.Error(w, "Test Failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, _ = w.Write([]byte("Test message sent! Check your Telegram."))
}
