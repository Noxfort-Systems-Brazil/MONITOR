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
// File: internal/transport/http/auth_helper.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
)

const sessionCookieName = "noxfort_session"

// ExtractSessionToken retrieves the session token from cookies or Authorization/session headers.
// Query string tokens are intentionally rejected to avoid leaking secrets in URLs and proxy logs.
func ExtractSessionToken(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if token := strings.TrimSpace(r.Header.Get("X-Session-Token")); token != "" {
		return token
	}
	return ""
}

// isTrustedProxyIP returns true when remoteIP is a loopback, private (e.g., Docker bridge), or explicitly trusted address.
func isTrustedProxyIP(remoteIP net.IP) bool {
	if remoteIP == nil {
		return false
	}
	if remoteIP.IsLoopback() || remoteIP.IsPrivate() {
		return true
	}
	if raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); raw != "" {
		for _, entry := range strings.Split(raw, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(remoteIP) {
				return true
			}
			if ip := net.ParseIP(entry); ip != nil && ip.Equal(remoteIP) {
				return true
			}
		}
	}
	return false
}

func parseRemoteIP(addr string) net.IP {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}
	return net.ParseIP(addr)
}

// ExtractClientIP determines the client's real IP address, honoring forwarded headers only from trusted proxies.
func ExtractClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	remoteIP := parseRemoteIP(r.RemoteAddr)
	if isTrustedProxyIP(remoteIP) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			for _, part := range strings.Split(xff, ",") {
				if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
					return ip.String()
				}
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			if ip := parseRemoteIP(xri); ip != nil {
				return ip.String()
			}
		}
	}
	if remoteIP != nil {
		return remoteIP.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// IsHTTPSRequest detects whether the client reached the server over HTTPS directly or via a trusted TLS proxy.
func IsHTTPSRequest(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("MONITOR_USE_HTTPS")), "true") {
		return true
	}
	if r != nil && isTrustedProxyIP(parseRemoteIP(r.RemoteAddr)) {
		return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
	}
	return false
}

// parseCredentials extracts username and password from form-encoded or JSON payloads.
func parseCredentials(r *http.Request) (string, string) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			username = strings.TrimSpace(req.Username)
			password = req.Password
		}
	}

	return username, password
}
