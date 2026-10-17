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
// File: internal/transport/http/template_cache.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"html/template"
	"net/http"
	"strings"
	"sync"

	"noxfort-monitor-server/internal/appdir"
)

// TemplateCache caches parsed HTML templates to eliminate per-request disk reads and parsing overhead.
type TemplateCache struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
}

var defaultTemplateCache = NewTemplateCache()

// NewTemplateCache initializes an in-memory template cache.
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*template.Template),
	}
}

// Get parses or retrieves a cached template constructed from the provided relative file paths.
func (c *TemplateCache) Get(files ...string) (*template.Template, error) {
	key := strings.Join(files, "|")

	c.mu.RLock()
	tmpl, exists := c.templates[key]
	c.mu.RUnlock()
	if exists {
		return tmpl, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock
	if tmpl, exists := c.templates[key]; exists {
		return tmpl, nil
	}

	fullPaths := make([]string, len(files))
	for i, f := range files {
		fullPaths[i] = appdir.Path(f)
	}

	parsed, err := template.ParseFiles(fullPaths...)
	if err != nil {
		return nil, err
	}

	c.templates[key] = parsed
	return parsed, nil
}

// Render executes the cached template with data to the writer.
func (c *TemplateCache) Render(w http.ResponseWriter, data interface{}, files ...string) error {
	tmpl, err := c.Get(files...)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// RenderTemplate is a package-level helper that renders using the default cache.
func RenderTemplate(w http.ResponseWriter, data interface{}, files ...string) error {
	return defaultTemplateCache.Render(w, data, files...)
}

// RenderPage renders a standard application page wrapped in web/templates/layout.html.
func RenderPage(w http.ResponseWriter, pageFile string, data interface{}) error {
	return RenderTemplate(w, data, "web/templates/layout.html", pageFile)
}

// GetTemplate is a package-level helper that retrieves a parsed template from the default cache.
func GetTemplate(files ...string) (*template.Template, error) {
	return defaultTemplateCache.Get(files...)
}
