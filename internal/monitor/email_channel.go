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
// File: internal/monitor/email_channel.go
// Author: Gabriel Moraes
// Date: October 2026

package monitor

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// SMTPSenderFunc represents the low-level SMTP send function, allowing test mocking.
type SMTPSenderFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// EmailChannel handles email formatting and SMTP delivery.
type EmailChannel struct {
	sendMail         SMTPSenderFunc
	customSendMail   bool
}

// NewEmailChannel creates a new EmailChannel using standard net/smtp.SendMail.
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{sendMail: smtp.SendMail}
}

// NewEmailChannelWithSender creates an EmailChannel with a custom SMTP sender (for testing).
func NewEmailChannelWithSender(sender SMTPSenderFunc) *EmailChannel {
	return &EmailChannel{sendMail: sender, customSendMail: true}
}

// Name returns the channel name.
func (c *EmailChannel) Name() string {
	return "email"
}

// Recipient extracts the email destination address from the contact.
func (c *EmailChannel) Recipient(contact *domain.Contact) string {
	if contact == nil {
		return ""
	}
	return contact.Email
}

// BuildSubject constructs a standardized alert email subject.
func (c *EmailChannel) BuildSubject(identifier string, event *domain.IncomingEvent) string {
	return fmt.Sprintf("[%s] %s - %s: %s", event.Level, event.Category, identifier, event.Message)
}

// BuildBody formats the plain-text body of the incident email.
func (c *EmailChannel) BuildBody(identifier string, event *domain.IncomingEvent) string {
	return fmt.Sprintf(
		"NOXFORT MONITOR - RELATÓRIO DE INCIDENTE\n"+
			"--------------------------------------------------\n"+
			"Categoria:    %s\n"+
			"Sistema:      %s\n"+
			"Gravidade:    %s\n"+
			"Data/Hora:    %s\n"+
			"--------------------------------------------------\n"+
			"\n"+
			"MENSAGEM:\n"+
			"%s\n",
		event.Category,
		identifier,
		event.Level,
		event.OccurredAt.Format("02/01/2006 15:04:05"),
		event.Message,
	)
}

// Send formats and sends an alert email to the contact if SMTP and recipient email are configured.
func (c *EmailChannel) Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error {
	if settings == nil || !settings.Enabled || contact == nil || contact.Email == "" {
		return nil
	}

	subject := c.BuildSubject(identifier, event)
	body := c.BuildBody(identifier, event)

	return c.transmit(settings, contact.Email, subject, body)
}

// Test sends a verification email to check SMTP connectivity.
func (c *EmailChannel) Test(settings *domain.Settings, to string) error {
	subject := "Noxfort Monitor: Test Connection"
	body := "This is a test email to verify your SMTP configuration."
	return c.transmit(settings, to, subject, body)
}

func sanitizeEmailHeader(v string) string {
	r := strings.NewReplacer("\r", "", "\n", "")
	return strings.TrimSpace(r.Replace(v))
}

func (c *EmailChannel) transmit(settings *domain.Settings, to, subject, body string) error {
	cleanTo := sanitizeEmailHeader(to)
	cleanFrom := sanitizeEmailHeader(settings.SMTPFrom)
	if cleanFrom == "" {
		cleanFrom = sanitizeEmailHeader(settings.SMTPUser)
	}
	cleanSubject := sanitizeEmailHeader(subject)

	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", cleanTo, cleanFrom, cleanSubject, body))

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)
	if settings.SMTPPort == 465 && !c.customSendMail {
		return sendImplicitTLS(addr, settings.SMTPHost, auth, cleanFrom, []string{cleanTo}, msg)
	}
	return c.sendMail(addr, auth, cleanFrom, []string{cleanTo}, msg)
}

func sendImplicitTLS(addr, serverName string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, serverName)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return client.Quit()
}
