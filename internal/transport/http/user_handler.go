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
// File: internal/transport/http/user_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/security"
)

// UserManagementService defines user account operations required by HTTP transport.
type UserManagementService interface {
	Register(username, password string) (*domain.User, error)
	RegisterWithRole(username, password, role string) (*domain.User, error)
	ListUsers() ([]*domain.User, error)
	DeleteUser(username string) error
}

// UserAuditService extends UserManagementService with actor/IP-attributed audit actions.
type UserAuditService interface {
	RegisterWithRoleByActor(actor, username, password, role, ip string) (*domain.User, error)
	DeleteUserByActor(actor, username, ip string) error
}

// UserAccountEditor extends UserManagementService with password reset, role editing, and lockout unlocking.
type UserAccountEditor interface {
	ChangePassword(actorUsername, targetUsername, currentPassword, newPassword, clientIP string) error
	UpdateUserRole(actorUsername, targetUsername, newRole, clientIP string) error
	UnlockAccount(actorUsername, targetUsername, clientIP string) error
}

// SessionValidator verifies active sessions and user roles.
type SessionValidator interface {
	ValidateSession(token string) (string, string, bool)
	RevokeSession(token string)
}

// UserHandler manages user accounts and operator administration.
type UserHandler struct {
	userService      UserManagementService
	sessionValidator SessionValidator
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(userService UserManagementService, sessionValidator SessionValidator) *UserHandler {
	return &UserHandler{
		userService:      userService,
		sessionValidator: sessionValidator,
	}
}

// GetSessionUser checks the request cookie and headers, returning username, role, and validity.
func (h *UserHandler) GetSessionUser(r *http.Request) (string, string, bool) {
	token := ExtractSessionToken(r)
	if token == "" {
		return "", "", false
	}
	return h.sessionValidator.ValidateSession(token)
}

// HandleCreateUser registers a new user (defaulting to OPERATOR). Only authenticated ADMIN users can perform this.
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		writeForbiddenResponse(w, r, "Acesso não autorizado: apenas o Administrador pode cadastrar novos usuários")
		return
	}

	username, password := parseCredentials(r)

	targetRole := strings.TrimSpace(r.FormValue("role"))
	if targetRole != domain.RoleAdmin && targetRole != domain.RoleOperator {
		targetRole = domain.RoleOperator
	}

	w.Header().Set("Content-Type", "application/json")

	var (
		user *domain.User
		err  error
	)
	if auditSvc, ok := h.userService.(UserAuditService); ok {
		user, err = auditSvc.RegisterWithRoleByActor(currentUser, username, password, targetRole, ExtractClientIP(r))
	} else {
		user, err = h.userService.RegisterWithRole(username, password, targetRole)
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": user.Username,
		"role":     user.Role,
	})
}

// HandleRegister is an alias for HandleCreateUser, requiring admin privileges.
func (h *UserHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	h.HandleCreateUser(w, r)
}

// HandleList returns all registered users in JSON (ADMIN only).
func (h *UserHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	_, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		writeForbiddenResponse(w, r, "Acesso não autorizado")
		return
	}

	users, err := h.userService.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

// HandleDelete removes an account (ADMIN only). Only the superuser cannot be deleted.
func (h *UserHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUsername, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		writeForbiddenResponse(w, r, "Acesso não autorizado: privilégios de administrador necessários")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Nome de usuário é obrigatório",
		})
		return
	}

	isSelfDelete := strings.EqualFold(username, currentUsername)

	var deleteErr error
	if auditSvc, ok := h.userService.(UserAuditService); ok {
		deleteErr = auditSvc.DeleteUserByActor(currentUsername, username, ExtractClientIP(r))
	} else {
		deleteErr = h.userService.DeleteUser(username)
	}
	if deleteErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": deleteErr.Error(),
		})
		return
	}

	if isSelfDelete {
		if token := ExtractSessionToken(r); token != "" {
			h.sessionValidator.RevokeSession(token)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		log.Printf("[USERS] User '%s' deleted their own account. Session terminated.", username)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"self_deleted": true,
			"message":      "Sua própria conta foi excluída com sucesso. Sessão encerrada.",
		})
		return
	}

	log.Printf("[USERS] User '%s' removed by Administrator '%s'.", username, currentUsername)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Usuário removido com sucesso",
	})
}

// HandleChangePassword lets admins reset any account password or authenticated users update their own password.
func (h *UserHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser, role, isAuth := h.GetSessionUser(r)
	if !isAuth {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	editor, ok := h.userService.(UserAccountEditor)
	if !ok {
		http.Error(w, `{"success":false,"message":"Alteração de senha indisponível"}`, http.StatusNotImplemented)
		return
	}

	targetUsername := strings.TrimSpace(r.FormValue("username"))
	if targetUsername == "" {
		targetUsername = currentUser
	}
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")

	if role != domain.RoleAdmin && !strings.EqualFold(targetUsername, currentUser) {
		writeForbiddenResponse(w, r, "Acesso restrito: apenas o Administrador pode redefinir a senha de outros usuários.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := editor.ChangePassword(currentUser, targetUsername, currentPassword, newPassword, ExtractClientIP(r)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Senha atualizada com sucesso. Sessões anteriores desta conta foram encerradas.",
	})
}

// HandleUpdateRole promotes or demotes a user's RBAC role (ADMIN only).
func (h *UserHandler) HandleUpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		writeForbiddenResponse(w, r, "Acesso não autorizado: apenas o Administrador pode alterar perfis.")
		return
	}

	editor, ok := h.userService.(UserAccountEditor)
	if !ok {
		http.Error(w, `{"success":false,"message":"Alteração de perfil indisponível"}`, http.StatusNotImplemented)
		return
	}

	targetUsername := strings.TrimSpace(r.FormValue("username"))
	newRole := strings.TrimSpace(r.FormValue("role"))

	w.Header().Set("Content-Type", "application/json")
	if err := editor.UpdateUserRole(currentUser, targetUsername, newRole, ExtractClientIP(r)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Perfil atualizado com sucesso.",
	})
}

// HandleUnlockUser clears brute-force lockout counters for a specific user (ADMIN only).
func (h *UserHandler) HandleUnlockUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser, role, isAuth := h.GetSessionUser(r)
	if !isAuth || role != domain.RoleAdmin {
		writeForbiddenResponse(w, r, "Acesso não autorizado: privilégios de administrador necessários.")
		return
	}

	editor, ok := h.userService.(UserAccountEditor)
	if !ok {
		http.Error(w, `{"success":false,"message":"Desbloqueio indisponível"}`, http.StatusNotImplemented)
		return
	}

	targetUsername := strings.TrimSpace(r.FormValue("username"))
	w.Header().Set("Content-Type", "application/json")
	if err := editor.UnlockAccount(currentUser, targetUsername, ExtractClientIP(r)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Bloqueio de login removido com sucesso.",
	})
}

// ServePage renders the /users account management page (ADMIN only).
func (h *UserHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	username, role, isAuth := h.GetSessionUser(r)
	if !isAuth {
		Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if role != domain.RoleAdmin {
		if _, _, ok := SessionFromContext(r); !ok {
			r = withSessionContext(r, username, role)
		}
		writeForbiddenResponse(w, r, "Acesso restrito: Apenas o Administrador pode acessar esta página.")
		return
	}

	users, err := h.userService.ListUsers()
	if err != nil {
		log.Printf("[USERS] Error listing users: %v", err)
	}

	data := map[string]interface{}{
		"CurrentUser": username,
		"Superuser":   security.SuperuserUsername,
		"Role":        role,
		"IsAdmin":     true,
		"Users":       users,
	}

	if err := RenderPageWithRequest(w, r, "web/templates/users.html", data); err != nil {
		log.Printf("[USERS] Template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
