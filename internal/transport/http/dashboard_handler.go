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
// File: internal/transport/http/dashboard_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// DashboardDeviceView represents a single monitored device row with lifecycle-aware status.
type DashboardDeviceView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	LastSeen   string `json:"last_seen"`
	Status     string `json:"status"`
	BadgeClass string `json:"badge_class"`
	RowClass   string `json:"row_class"`
}

// DashboardHandler manages the main overview page.
type DashboardHandler struct {
	deviceRepo    domain.DeviceRepository
	telemetryRepo domain.TelemetryRepository
}

// NewDashboardHandler creates the handler with necessary dependencies.
func NewDashboardHandler(dRepo domain.DeviceRepository, tRepo domain.TelemetryRepository) *DashboardHandler {
	return &DashboardHandler{
		deviceRepo:    dRepo,
		telemetryRepo: tRepo,
	}
}

func classifyDeviceStatus(d domain.Device, threshold time.Duration) (status, badgeClass, rowClass, lastSeenStr string) {
	if !d.LastSeen.IsZero() {
		lastSeenStr = d.LastSeen.Format("15:04:05 02/01/2006")
	} else {
		lastSeenStr = "-"
	}

	switch {
	case d.Status == "PENDING":
		return "PENDING", "bg-warning text-dark", "table-warning", lastSeenStr
	case !d.Enabled:
		return "PAUSED", "bg-secondary", "table-secondary", lastSeenStr
	case d.LastSeen.IsZero():
		return "WAITING", "bg-info text-dark", "table-dark", lastSeenStr
	case time.Since(d.LastSeen) > threshold:
		return "OFFLINE", "bg-danger", "table-danger", lastSeenStr
	default:
		return "ONLINE", "bg-success", "table-success", lastSeenStr
	}
}

func (h *DashboardHandler) buildSnapshot() ([]DashboardDeviceView, []domain.IncomingEvent, error) {
	devices, err := h.deviceRepo.GetAllDevices()
	if err != nil {
		return nil, nil, err
	}

	incidents, err := h.telemetryRepo.GetRecentIncidents(1000)
	if err != nil {
		log.Printf("[DASHBOARD] Error fetching incidents: %v", err)
		incidents = []domain.IncomingEvent{}
	}

	threshold := 5 * time.Minute
	viewData := make([]DashboardDeviceView, 0, len(devices))
	for _, d := range devices {
		status, badgeClass, rowClass, lastSeenStr := classifyDeviceStatus(d, threshold)
		viewData = append(viewData, DashboardDeviceView{
			ID:         d.ID,
			Name:       d.Name,
			Identifier: d.Identifier,
			LastSeen:   lastSeenStr,
			Status:     status,
			BadgeClass: badgeClass,
			RowClass:   rowClass,
		})
	}

	return viewData, incidents, nil
}

// ServePage renders the main dashboard HTML.
func (h *DashboardHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	viewData, incidents, err := h.buildSnapshot()
	if err != nil {
		log.Printf("[DASHBOARD] Error fetching devices: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "System Overview",
		"Devices":         viewData,
		"RecentIncidents": incidents,
		"Now":             time.Now().Format(time.RFC3339),
	}

	if err := RenderPageWithRequest(w, r, "web/templates/dashboard.html", data); err != nil {
		log.Printf("[DASHBOARD] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}
}

// HandleSummary returns the live device and recent incident snapshot as JSON for periodic UI updates.
func (h *DashboardHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	viewData, incidents, err := h.buildSnapshot()
	if err != nil {
		http.Error(w, `{"error":"Failed to load dashboard summary"}`, http.StatusInternalServerError)
		return
	}

	type incidentJSON struct {
		Time     string `json:"time"`
		Level    string `json:"level"`
		Origin   string `json:"origin"`
		Message  string `json:"message"`
		Category string `json:"category"`
	}
	incOut := make([]incidentJSON, 0, len(incidents))
	for _, inc := range incidents {
		incOut = append(incOut, incidentJSON{
			Time:     inc.OccurredAt.Format("15:04:05 02/01"),
			Level:    string(inc.Level),
			Origin:   inc.Origin,
			Message:  inc.Message,
			Category: string(inc.Category),
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"now":              time.Now().Format(time.RFC3339),
		"devices":          viewData,
		"recent_incidents": incOut,
	})
}
