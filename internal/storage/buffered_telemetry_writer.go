// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/buffered_telemetry_writer.go
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: In-memory batch ingestion queue and asynchronous flusher for telemetry events.

package storage

import (
	"log"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// BufferedTelemetryWriter collects events in an in-memory queue and flushes them in batches.
type BufferedTelemetryWriter struct {
	repo          *TelemetryRepository
	queue         chan TelemetryRecord
	flushInterval time.Duration
	batchSize     int
	stopChan      chan struct{}
	doneChan      chan struct{}
	mu            sync.Mutex
	closed        bool
}

// NewBufferedTelemetryWriter starts a background batch ingestion worker.
func NewBufferedTelemetryWriter(repo *TelemetryRepository, batchSize int, flushInterval time.Duration) *BufferedTelemetryWriter {
	if batchSize <= 0 {
		batchSize = 50
	}
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond
	}

	w := &BufferedTelemetryWriter{
		repo:          repo,
		queue:         make(chan TelemetryRecord, batchSize*4),
		flushInterval: flushInterval,
		batchSize:     batchSize,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}
	go w.worker()
	return w
}

// Enqueue adds a telemetry event to the buffered batch writer.
// If the buffer is full, it immediately falls back to synchronous write to avoid dropping data.
func (w *BufferedTelemetryWriter) Enqueue(identifier string, event *domain.IncomingEvent) bool {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return false
	}
	w.mu.Unlock()

	select {
	case w.queue <- TelemetryRecord{Identifier: identifier, Event: event}:
		return true
	default:
		// Queue full fallback
		_ = w.repo.SaveEvent(identifier, event)
		return true
	}
}

func (w *BufferedTelemetryWriter) worker() {
	defer close(w.doneChan)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]TelemetryRecord, 0, w.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := w.repo.SaveEventsBatch(batch); err != nil {
			log.Printf("[STORAGE] Buffered write error: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-w.stopChan:
			for {
				select {
				case item := <-w.queue:
					batch = append(batch, item)
					if len(batch) >= w.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case item := <-w.queue:
			batch = append(batch, item)
			if len(batch) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close gracefully flushes all queued items and terminates the worker.
func (w *BufferedTelemetryWriter) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stopChan)
	<-w.doneChan
}
