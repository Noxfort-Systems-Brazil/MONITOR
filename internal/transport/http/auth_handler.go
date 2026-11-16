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
// File: internal/transport/http/auth_handler.go
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

// AuthLockoutService extends AuthService with brute-force failure metadata and client IP tracking.
type AuthLockoutService interface {
	AuthenticateWithAttempt(username, password, clientIP string) (*domain.User, domain.AuthAttemptResult, error)
}

// AuthService defines the authentication and session management contract required by HTTP transport.
type AuthService interface {
	Authenticate(username, password string) (*domain.User, error)
	CreateSession(username, role string) string
	ValidateSession(token string) (string, string, bool)
	RevokeSession(token string)
}

// AuthHandler manages authentication, login, logout, and session state.
type AuthHandler struct {
	authService AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// ExtractToken retrieves the session token from cookies, Authorization headers, or custom headers.
func (h *AuthHandler) ExtractToken(r *http.Request) string {
	return ExtractSessionToken(r)
}

// GetSessionUser checks the request cookie and headers, returning the authenticated username, role, and true if valid.
func (h *AuthHandler) GetSessionUser(r *http.Request) (string, string, bool) {
	token := ExtractSessionToken(r)
	if token == "" {
		return "", "", false
	}
	return h.authService.ValidateSession(token)
}

// ServeLogin renders the login page.
func (h *AuthHandler) ServeLogin(w http.ResponseWriter, r *http.Request) {
	if _, _, isAuth := h.GetSessionUser(r); isAuth {
		Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := RenderTemplate(w, nil, "web/templates/login.html"); err != nil {
		log.Printf("[AUTH] Template error (login): %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// ServeRegister redirects unauthenticated visitors to /login, or authenticated admins to /users.
func (h *AuthHandler) ServeRegister(w http.ResponseWriter, r *http.Request) {
	_, role, isAuth := h.GetSessionUser(r)
	if isAuth && role == domain.RoleAdmin {
		Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleLogin authenticates credentials and sets an HTTP-only session cookie.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	username, password := parseCredentials(r)
	clientIP := ExtractClientIP(r)

	w.Header().Set("Content-Type", "application/json")

	var user *domain.User
	var attempt domain.AuthAttemptResult
	var err error
	if lockoutSvc, ok := h.authService.(AuthLockoutService); ok {
		user, attempt, err = lockoutSvc.AuthenticateWithAttempt(username, password, clientIP)
	} else {
		user, err = h.authService.Authenticate(username, password)
	}

	if err != nil {
		statusCode := http.StatusUnauthorized
		if attempt.IsLockdown {
			statusCode = http.StatusTooManyRequests
		}
		w.WriteHeader(statusCode)
		resp := map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
		if attempt.MaxAttempts > 0 {
			resp["failed_attempts"] = attempt.FailedAttempts
			resp["max_attempts"] = attempt.MaxAttempts
			resp["is_lockdown"] = attempt.IsLockdown
			if attempt.RetryAfterSeconds > 0 {
				resp["retry_after_seconds"] = attempt.RetryAfterSeconds
			}
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	token := h.authService.CreateSession(user.Username, user.Role)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   IsHTTPSRequest(r),
		SameSite: http.SameSiteLaxMode,
	})

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"token":    token,
		"username": user.Username,
		"role":     user.Role,
	})
}

// HandleLogout clears the user session and redirects to /login.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if token := ExtractSessionToken(r); token != "" {
		h.authService.RevokeSession(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   IsHTTPSRequest(r),
		SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}

	Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleStatus returns JSON info about the current session.
func (h *AuthHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	username, role, isAuth := h.GetSessionUser(r)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": isAuth,
		"username":      username,
		"role":          role,
	})
}
