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
