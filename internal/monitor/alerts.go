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
// File: internal/monitor/alerts.go
// Author: Gabriel Moraes
// Date: October 2026

package monitor

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/metrics"
)

// AlertService orchestrates the retrieval of settings/contacts and dispatches alerts via configured channels.
type AlertService struct {
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	channels     []NotificationChannel
	retryDelays  []time.Duration
	wg           sync.WaitGroup
	closing      atomic.Bool
}

// SetAuditRepository attaches the audit repository for recording notification delivery logs.
func (s *AlertService) SetAuditRepository(repo domain.AuditRepository) {
	s.auditRepo = repo
}

// SetRetryDelays configures backoff durations between failed dispatch attempts (useful for deterministic testing).
func (s *AlertService) SetRetryDelays(delays ...time.Duration) {
	s.retryDelays = delays
}

// Shutdown marks the alert service as closing and waits for in-flight alert dispatches until ctx expires.
func (s *AlertService) Shutdown(ctx context.Context) error {
	s.closing.Store(true)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewAlertService creates a new instance of the AlertService with default or injected channels.
func NewAlertService(cRepo domain.ContactRepository, sRepo domain.SettingsRepository, channels ...NotificationChannel) *AlertService {
	if len(channels) == 0 {
		channels = []NotificationChannel{NewEmailChannel(), NewTelegramChannel()}
	}

	return &AlertService{
		contactRepo:  cRepo,
		settingsRepo: sRepo,
		channels:     channels,
		retryDelays:  []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond},
	}
}

// NewAlertServiceWithChannels creates an AlertService with custom channels (useful for testing or extensions).
func NewAlertServiceWithChannels(
	cRepo domain.ContactRepository,
	sRepo domain.SettingsRepository,
	channels ...NotificationChannel,
) *AlertService {
	return NewAlertService(cRepo, sRepo, channels...)
}

// TriggerAlert processes an incoming event, evaluates routing rules, and dispatches to eligible contacts.
func (s *AlertService) TriggerAlert(identifier string, event *domain.IncomingEvent) {
	s.TriggerAlertWithTelemetryID(identifier, event, nil)
}

// TriggerAlertWithTelemetryID dispatches alerts while linking delivery logs to the persisted telemetry row ID.
func (s *AlertService) TriggerAlertWithTelemetryID(identifier string, event *domain.IncomingEvent, telemetryID *int64) {
	if event == nil || s.closing.Load() {
		return
	}

	// 1. Fetch Global Settings
	settings, err := s.settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[ALERTS] Failed to fetch settings: %v", err)
		return
	}

	// 2. Fetch Contacts
	contacts, err := s.contactRepo.GetAllContacts()
	if err != nil {
		log.Printf("[ALERTS] Failed to fetch contacts: %v", err)
		return
	}

	// 3. Smart Routing & Channel Dispatch
	sentCount := 0
	for _, contact := range contacts {
		target := contact
		if !IsEligible(&target, event) {
			continue
		}

		for _, ch := range s.channels {
			if s.closing.Load() {
				continue
			}
			s.wg.Add(1)
			go func(channel NotificationChannel, c domain.Contact, telID *int64) {
				defer s.wg.Done()
				err := s.sendWithRetry(channel, settings, &c, identifier, event)
				status := "SENT"
				errReason := ""
				if err != nil {
					status = "FAILED"
					errReason = err.Error()
					log.Printf("[ALERTS] Failed to send alert via %s to contact '%s': %v",
						channel.Name(), c.Name, err)
				}
				metrics.IncAlertDispatched(channel.Name(), status)

				if s.auditRepo != nil {
					recipient := channel.Recipient(&c)
					_ = s.auditRepo.SaveAlertDispatchLog(&domain.AlertDispatchLog{
						TelemetryID:  telID,
						Channel:      channel.Name(),
						Recipient:    recipient,
						Role:         c.Role,
						Status:       status,
						ErrorReason:  errReason,
						DispatchedAt: event.OccurredAt,
					})
				}
			}(ch, target, telemetryID)
		}
		sentCount++
	}

	if sentCount > 0 {
		log.Printf("[ALERTS] Dispatching incident '%s' to %d eligible recipients across %d channels.",
			event.Message, sentCount, len(s.channels))
	}
}

func (s *AlertService) sendWithRetry(
	channel NotificationChannel,
	settings *domain.Settings,
	contact *domain.Contact,
	identifier string,
	event *domain.IncomingEvent,
) error {
	var err error
	attempts := 1 + len(s.retryDelays)
	for i := 0; i < attempts; i++ {
		err = channel.Send(settings, contact, identifier, event)
		if err == nil {
			return nil
		}
		if i < len(s.retryDelays) && !s.closing.Load() {
			time.Sleep(s.retryDelays[i])
		}
	}
	return err
}
