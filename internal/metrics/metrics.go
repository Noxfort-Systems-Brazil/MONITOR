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
// File: internal/metrics/metrics.go
// Author: Gabriel Moraes
// Date: October 2026

package metrics

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// CounterEntry represents a labeled counter value exported to Prometheus.
type CounterEntry struct {
	LabelA string
	LabelB string
	Value  int64
}

type labeledCounters struct {
	mu     sync.RWMutex
	values map[string]*atomic.Int64
}

func newLabeledCounters() *labeledCounters {
	return &labeledCounters{values: make(map[string]*atomic.Int64)}
}

func (c *labeledCounters) inc(a, b string) {
	key := a + "\x00" + b
	c.mu.RLock()
	ptr, ok := c.values[key]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		ptr, ok = c.values[key]
		if !ok {
			ptr = &atomic.Int64{}
			c.values[key] = ptr
		}
		c.mu.Unlock()
	}
	ptr.Add(1)
}

func (c *labeledCounters) snapshot() []CounterEntry {
	c.mu.RLock()
	out := make([]CounterEntry, 0, len(c.values))
	for k, ptr := range c.values {
		parts := strings.SplitN(k, "\x00", 2)
		b := ""
		if len(parts) > 1 {
			b = parts[1]
		}
		out = append(out, CounterEntry{
			LabelA: parts[0],
			LabelB: b,
			Value:  ptr.Load(),
		})
	}
	c.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].LabelA != out[j].LabelA {
			return out[i].LabelA < out[j].LabelA
		}
		return out[i].LabelB < out[j].LabelB
	})
	return out
}

var (
	ingestedCounters = newLabeledCounters()
	rejectedCounters = newLabeledCounters()
	alertCounters    = newLabeledCounters()
	bufferDepth      atomic.Int64
)

// IncTelemetryIngested increments accepted telemetry event counters by transport and severity level.
func IncTelemetryIngested(transport, level string) {
	ingestedCounters.inc(strings.ToLower(strings.TrimSpace(transport)), strings.ToUpper(strings.TrimSpace(level)))
}

// IncTelemetryRejected increments rejected telemetry event counters by transport and failure reason.
func IncTelemetryRejected(transport, reason string) {
	rejectedCounters.inc(strings.ToLower(strings.TrimSpace(transport)), strings.ToLower(strings.TrimSpace(reason)))
}

// IncAlertDispatched increments alert delivery counters by channel and final status (SENT/FAILED).
func IncAlertDispatched(channel, status string) {
	alertCounters.inc(strings.ToLower(strings.TrimSpace(channel)), strings.ToUpper(strings.TrimSpace(status)))
}

// SetBufferDepth records the current telemetry write queue depth.
func SetBufferDepth(depth int64) {
	if depth < 0 {
		depth = 0
	}
	bufferDepth.Store(depth)
}

// SnapshotIngested returns a sorted snapshot of accepted telemetry counters.
func SnapshotIngested() []CounterEntry { return ingestedCounters.snapshot() }

// SnapshotRejected returns a sorted snapshot of rejected telemetry counters.
func SnapshotRejected() []CounterEntry { return rejectedCounters.snapshot() }

// SnapshotAlerts returns a sorted snapshot of dispatched alert counters.
func SnapshotAlerts() []CounterEntry { return alertCounters.snapshot() }

// GetBufferDepth returns the current telemetry buffer queue length.
func GetBufferDepth() int64 { return bufferDepth.Load() }
