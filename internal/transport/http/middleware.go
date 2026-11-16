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
// File: internal/transport/http/middleware.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
)

type contextKey string

const sessionContextKey contextKey = "noxfort_session_user"

type sessionUserContext struct {
	Username string
	Role     string
}

// withSessionContext attaches the authenticated username and role to the request context.
func withSessionContext(r *http.Request, username, role string) *http.Request {
	if r == nil || username == "" {
		return r
	}
	ctx := context.WithValue(r.Context(), sessionContextKey, sessionUserContext{
		Username: username,
		Role:     role,
	})
	return r.WithContext(ctx)
}

// SessionFromContext retrieves the authenticated username and role injected by AuthMiddleware.
func SessionFromContext(r *http.Request) (username, role string, ok bool) {
	if r == nil {
		return "", "", false
	}
	val, ok := r.Context().Value(sessionContextKey).(sessionUserContext)
	if !ok || val.Username == "" {
		return "", "", false
	}
	return val.Username, val.Role, true
}

// logSecurityAuditEvent records an administrative action attributed to the real authenticated user and IP.
func logSecurityAuditEvent(auditRepo domain.AuditRepository, r *http.Request, action, details string) {
	if auditRepo == nil {
		return
	}
	username, _, ok := SessionFromContext(r)
	if !ok || username == "" {
		username = "admin"
	}
	_ = auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
		Username:  username,
		Action:    action,
		Details:   details,
		IPAddress: ExtractClientIP(r),
		CreatedAt: time.Now(),
	})
}

// AuthSessionInspector verifies session credentials and can serve the login page.
type AuthSessionInspector interface {
	GetSessionUser(r *http.Request) (string, string, bool)
	ServeLogin(w http.ResponseWriter, r *http.Request)
}

// AuthMiddleware intercepts requests to enforce authentication and operator read-only restrictions (RBAC).
type AuthMiddleware struct {
	authInspector AuthSessionInspector
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
func NewAuthMiddleware(authInspector AuthSessionInspector) *AuthMiddleware {
	return &AuthMiddleware{authInspector: authInspector}
}

// isPublicPath determines if a path can be accessed without an active session.
func isPublicPath(path string) bool {
	return strings.HasPrefix(path, "/static/") ||
		path == "/login" ||
		path == "/api/auth/login" ||
		path == "/api/auth/status" ||
		path == "/api/telemetry" ||
		path == "/api/open-external" ||
		path == "/healthz" ||
		path == "/api/health" ||
		path == "/metrics" ||
		strings.HasPrefix(path, "/api/window/")
}

// isMutatingRoute checks if the request represents a state modification action restricted to administrators.
func isMutatingRoute(r *http.Request, path string) bool {
	if path == "/api/auth/logout" || path == "/api/users/password" || path == "/api/open-external" || strings.HasPrefix(path, "/api/window/") {
		return false
	}

	isWriteMethod := r.Method == http.MethodPost ||
		r.Method == http.MethodPut ||
		r.Method == http.MethodPatch ||
		r.Method == http.MethodDelete

	return isWriteMethod ||
		path == "/devices/delete" ||
		strings.HasPrefix(path, "/contacts/create") ||
		strings.HasPrefix(path, "/contacts/update") ||
		strings.HasPrefix(path, "/contacts/delete") ||
		strings.HasPrefix(path, "/settings/save") ||
		strings.HasPrefix(path, "/settings/test")
}

// wantsJSONResponse checks whether the caller expects a JSON error payload instead of an HTML page.
func wantsJSONResponse(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	if strings.Contains(accept, "application/json") {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
}

// resolveForbiddenBackURL chooses the safest Monitor page to return to when a restricted action fails.
func resolveForbiddenBackURL(path string) string {
	switch {
	case strings.HasPrefix(path, "/devices"):
		return "/devices"
	case strings.HasPrefix(path, "/contacts"):
		return "/contacts"
	case strings.HasPrefix(path, "/settings"):
		return "/settings"
	default:
		return "/"
	}
}

// writeForbiddenResponse writes a content-negotiated 403 Forbidden response so API callers
// receive structured JSON and WebKitGTK desktop navigations stay inside the Monitor HTML shell.
func writeForbiddenResponse(w http.ResponseWriter, r *http.Request, message string) {
	if wantsJSONResponse(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   message,
			"message": message,
		})
		return
	}

	accept := strings.ToLower(r.Header.Get("Accept"))
	if strings.HasPrefix(r.URL.Path, "/settings/test") && !strings.Contains(accept, "text/html") {
		http.Error(w, message, http.StatusForbidden)
		return
	}

	backURL := resolveForbiddenBackURL(r.URL.Path)
	pageData := map[string]interface{}{
		"Title":   "Acesso Restrito",
		"Message": message,
		"BackURL": backURL,
	}

	rendered, err := RenderPageToBufferWithRequest(r, "web/templates/forbidden.html", pageData)
	if err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(rendered)
		return
	}

	log.Printf("[RBAC] Failed to render forbidden.html template: %v", err)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="pt-BR"><head><meta charset="UTF-8"><title>Acesso Restrito · Noxfort Monitor</title></head><body style="background:#0d1117;color:#c9d1d9;font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;"><div style="max-width:480px;padding:24px;background:#161b22;border:1px solid #f0ad4e;border-radius:8px;"><h2>Acesso Restrito</h2><p>%s</p><p><a href="%s" style="color:#58a6ff;">Voltar</a> &middot; <a href="/" style="color:#58a6ff;">Ir para o Painel</a></p></div></body></html>`,
		html.EscapeString(message), html.EscapeString(backURL))
}

// Wrap wraps an existing http.Handler with authentication and RBAC checks.
func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 1. Allow public endpoints
		if isPublicPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		// Prevent browser from caching protected views in memory/disk bfcache
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")

		// 2. Verify active session
		username, role, isAuth := m.authInspector.GetSessionUser(r)
		if !isAuth {
			if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
					Secure:   IsHTTPSRequest(r),
					SameSite: http.SameSiteLaxMode,
				})
			}
			if strings.HasPrefix(path, "/api/") {
				http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			// When unauthenticated access hits root path, serve login directly
			if path == "/" {
				m.authInspector.ServeLogin(w, r)
				return
			}
			Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		r = withSessionContext(r, username, role)

		// 3. Enforce Admin-only write restrictions (Operators are read-only)
		// Also guard sensitive database backup endpoints that contain credential hashes and device tokens.
		if role != domain.RoleAdmin && (isMutatingRoute(r, path) || strings.HasPrefix(path, "/api/settings/database/backup")) {
			writeForbiddenResponse(w, r, "Acesso restrito: Operadores têm permissão somente leitura. Apenas o Administrador pode efetuar alterações.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
