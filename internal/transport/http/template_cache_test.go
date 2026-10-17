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
// File: internal/transport/http/template_cache_test.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"net/http/httptest"
	"sync"
	"testing"
)

func TestTemplateCache_GetAndRender(t *testing.T) {
	cache := NewTemplateCache()

	// 1. Render login template
	w := httptest.NewRecorder()
	err := cache.Render(w, nil, "web/templates/login.html")
	if err != nil {
		t.Fatalf("Failed to render login template: %v", err)
	}

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty rendered body")
	}

	// 2. Second access should hit cache
	tmpl1, err := cache.Get("web/templates/login.html")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}
	tmpl2, err := cache.Get("web/templates/login.html")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}
	if tmpl1 != tmpl2 {
		t.Error("Expected identical template pointer from cache")
	}
}

func TestTemplateCache_ConcurrentAccess(t *testing.T) {
	cache := NewTemplateCache()
	var wg sync.WaitGroup
	const goroutines = 20

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			_ = cache.Render(w, map[string]interface{}{"Title": "Test"}, "web/templates/layout.html", "web/templates/dashboard.html")
		}()
	}
	wg.Wait()
}

func TestTemplateCache_NonExistentFile(t *testing.T) {
	cache := NewTemplateCache()
	_, err := cache.Get("web/templates/does_not_exist_xyz.html")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
