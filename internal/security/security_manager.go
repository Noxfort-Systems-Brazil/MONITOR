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
// File: internal/security/security_manager.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// SecurityManager coordinates user registration, authentication, and session management.
type SecurityManager struct {
	userRepo     domain.UserRepository
	hasher       PasswordHasher
	sessionStore SessionStore
	auditRepo    domain.AuditRepository
	tracker      *LoginAttemptTracker
}

// NewSecurityManager creates a new SecurityManager instance with default PBKDF2 hasher and memory store.
func NewSecurityManager(userRepo domain.UserRepository) *SecurityManager {
	return NewSecurityManagerWithCustom(userRepo, defaultHasher, NewMemorySessionStore(SessionTTL))
}

// NewPersistentSecurityManager creates a SecurityManager backed by a persistent SessionStore with the default PBKDF2 hasher.
func NewPersistentSecurityManager(userRepo domain.UserRepository, sessionStore SessionStore) *SecurityManager {
	return NewSecurityManagerWithCustom(userRepo, defaultHasher, sessionStore)
}

// NewSecurityManagerWithCustom creates a SecurityManager with injected PasswordHasher and SessionStore.
func NewSecurityManagerWithCustom(userRepo domain.UserRepository, hasher PasswordHasher, sessionStore SessionStore) *SecurityManager {
	if hasher == nil {
		hasher = defaultHasher
	}
	if sessionStore == nil {
		sessionStore = NewMemorySessionStore(SessionTTL)
	}
	return &SecurityManager{
		userRepo:     userRepo,
		hasher:       hasher,
		sessionStore: sessionStore,
		tracker:      NewLoginAttemptTracker(DefaultMaxFailedAttempts, DefaultAttemptWindow, DefaultLockoutDuration),
	}
}

// SetAuditRepository sets the audit repository for recording security events.
func (sm *SecurityManager) SetAuditRepository(repo domain.AuditRepository) {
	sm.auditRepo = repo
}

func (sm *SecurityManager) logAudit(username, action, details, ip string) {
	if sm.auditRepo != nil {
		_ = sm.auditRepo.SaveSecurityAuditLog(&domain.SecurityAuditLog{
			Username:  username,
			Action:    action,
			Details:   details,
			IPAddress: ip,
			CreatedAt: time.Now(),
		})
	}
}

// EnsureSuperuser guarantees that the designated superuser account exists with ADMIN role.
func (sm *SecurityManager) EnsureSuperuser() error {
	return BootstrapSuperuser(sm.userRepo, sm.hasher)
}

// HasAdmin checks whether there is already an administrator registered in the system.
func (sm *SecurityManager) HasAdmin() (bool, error) {
	count, err := sm.userRepo.CountAdmins()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RegisterWithRole creates a new user account with an explicit role (ADMIN or OPERATOR).
func (sm *SecurityManager) RegisterWithRole(username, password, role string) (*domain.User, error) {
	return sm.RegisterWithRoleByActor(username, username, password, role, "")
}

// RegisterWithRoleByActor creates a new user account while attributing the audit log to the acting administrator.
func (sm *SecurityManager) RegisterWithRoleByActor(actor, username, password, role, ip string) (*domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("Nome de usuário e senha são obrigatórios")
	}

	if err := ValidatePasswordComplexity(password); err != nil {
		return nil, err
	}

	existing, _ := sm.userRepo.GetByUsername(username)
	if existing != nil {
		return nil, errors.New("Nome de usuário já está em uso")
	}

	if role != domain.RoleAdmin && role != domain.RoleOperator {
		role = domain.RoleOperator
	}

	hash, err := sm.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("falha ao criptografar senha: %w", err)
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now(),
	}

	if err := sm.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("falha ao salvar usuário: %w", err)
	}

	auditActor := strings.TrimSpace(actor)
	if auditActor == "" {
		auditActor = username
	}
	sm.logAudit(auditActor, "USER_CREATE", fmt.Sprintf("Usuário '%s' cadastrado com perfil %s", username, role), ip)
	log.Printf("[SECURITY] Novo usuário '%s' cadastrado com perfil %s por '%s'.", username, role, auditActor)
	return user, nil
}

// Register creates a new user account following the business rule:
// - If no admin exists in the system, the user is created as ADMIN.
// - All subsequent registrations become OPERATOR.
func (sm *SecurityManager) Register(username, password string) (*domain.User, error) {
	adminCount, err := sm.userRepo.CountAdmins()
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar administradores: %w", err)
	}

	role := domain.RoleOperator
	if adminCount == 0 {
		role = domain.RoleAdmin
		log.Printf("[SECURITY] Primeiro usuário do sistema '%s' definido como ADMINISTRADOR.", username)
	}

	return sm.RegisterWithRole(username, password, role)
}


// Authenticate verifies user credentials against the database.
func (sm *SecurityManager) Authenticate(username, password string) (*domain.User, error) {
	user, _, err := sm.AuthenticateWithAttempt(username, password, "")
	return user, err
}

// AuthenticateWithAttempt verifies credentials while enforcing brute-force rate limiting and IP-aware auditing.
func (sm *SecurityManager) AuthenticateWithAttempt(username, password, clientIP string) (*domain.User, domain.AuthAttemptResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		sm.logAudit(username, "AUTH_LOGIN_FAILED", "Tentativa de login com campos vazios", clientIP)
		return nil, domain.AuthAttemptResult{}, errors.New("Usuário e senha são obrigatórios")
	}

	if sm.tracker != nil {
		if status, locked := sm.tracker.Check(username, clientIP); locked {
			sm.logAudit(username, "AUTH_LOGIN_LOCKED", fmt.Sprintf("Login bloqueado por excesso de tentativas (%d/%d)", status.FailedAttempts, status.MaxAttempts), clientIP)
			return nil, status, fmt.Errorf("Muitas tentativas inválidas. Tente novamente em %d segundos.", status.RetryAfterSeconds)
		}
	}

	user, err := sm.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		var status domain.AuthAttemptResult
		if sm.tracker != nil {
			status = sm.tracker.RecordFailure(username, clientIP)
		}
		sm.logAudit(username, "AUTH_LOGIN_FAILED", "Usuário não encontrado", clientIP)
		return nil, status, errors.New("Usuário ou senha incorretos")
	}

	if !sm.hasher.Verify(password, user.PasswordHash) {
		var status domain.AuthAttemptResult
		if sm.tracker != nil {
			status = sm.tracker.RecordFailure(username, clientIP)
		}
		sm.logAudit(username, "AUTH_LOGIN_FAILED", fmt.Sprintf("Senha incorreta (tentativa %d/%d)", status.FailedAttempts, status.MaxAttempts), clientIP)
		return nil, status, errors.New("Usuário ou senha incorretos")
	}

	if sm.tracker != nil {
		sm.tracker.RecordSuccess(username, clientIP)
	}

	sm.logAudit(user.Username, "AUTH_LOGIN_SUCCESS", fmt.Sprintf("Login efetuado com perfil %s", user.Role), clientIP)
	log.Printf("[SECURITY] Usuário '%s' (%s) logado com sucesso.", user.Username, user.Role)
	return user, domain.AuthAttemptResult{MaxAttempts: DefaultMaxFailedAttempts}, nil
}

// CreateSession generates a new session token via the configured SessionStore.
func (sm *SecurityManager) CreateSession(username, role string) string {
	return sm.sessionStore.CreateSession(username, role)
}

// ValidateSession verifies if a session token is active and valid via the configured SessionStore.
func (sm *SecurityManager) ValidateSession(token string) (string, string, bool) {
	return sm.sessionStore.ValidateSession(token)
}

// RevokeSession invalidates an active session token via the configured SessionStore.
func (sm *SecurityManager) RevokeSession(token string) {
	sm.sessionStore.RevokeSession(token)
}

func (sm *SecurityManager) revokeUserSessions(username string) {
	if revoker, ok := sm.sessionStore.(interface{ RevokeUserSessions(string) }); ok {
		revoker.RevokeUserSessions(username)
	}
}

// ListUsers returns all users in the system.
func (sm *SecurityManager) ListUsers() ([]*domain.User, error) {
	return sm.userRepo.List()
}

// ChangePassword updates an account's password after enforcing password policy, revoking existing sessions on change.
// Non-admin callers changing their own password must supply the valid currentPassword.
func (sm *SecurityManager) ChangePassword(actorUsername, targetUsername, currentPassword, newPassword, clientIP string) error {
	targetUsername = strings.TrimSpace(targetUsername)
	actorUsername = strings.TrimSpace(actorUsername)
	if targetUsername == "" || newPassword == "" {
		return errors.New("Usuário e nova senha são obrigatórios")
	}

	if err := ValidatePasswordComplexity(newPassword); err != nil {
		return err
	}

	target, err := sm.userRepo.GetByUsername(targetUsername)
	if err != nil || target == nil {
		return errors.New("Usuário não encontrado")
	}

	isSelf := strings.EqualFold(actorUsername, targetUsername)
	if isSelf && currentPassword != "" {
		if !sm.hasher.Verify(currentPassword, target.PasswordHash) {
			return errors.New("Senha atual incorreta")
		}
	} else {
		actor, actorErr := sm.userRepo.GetByUsername(actorUsername)
		if actorErr != nil || actor == nil || actor.Role != domain.RoleAdmin {
			if !isSelf || currentPassword == "" {
				return errors.New("Senha atual obrigatória ou permissão de administrador necessária")
			}
		}
	}

	hash, err := sm.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("falha ao criptografar nova senha: %w", err)
	}

	if err := sm.userRepo.UpdatePasswordHash(targetUsername, hash); err != nil {
		return fmt.Errorf("falha ao atualizar senha: %w", err)
	}

	sm.revokeUserSessions(targetUsername)
	if sm.tracker != nil {
		sm.tracker.UnlockUser(targetUsername)
	}

	auditActor := actorUsername
	if auditActor == "" {
		auditActor = targetUsername
	}
	sm.logAudit(auditActor, "USER_PASSWORD_CHANGED", fmt.Sprintf("Senha atualizada para '%s'", targetUsername), clientIP)
	return nil
}

// UpdateUserRole promotes or demotes an account between ADMIN and OPERATOR while safeguarding superuser and last admin.
func (sm *SecurityManager) UpdateUserRole(actorUsername, targetUsername, newRole, clientIP string) error {
	targetUsername = strings.TrimSpace(targetUsername)
	newRole = strings.TrimSpace(newRole)
	if newRole != domain.RoleAdmin && newRole != domain.RoleOperator {
		return errors.New("Perfil inválido (deve ser ADMIN ou OPERATOR)")
	}

	target, err := sm.userRepo.GetByUsername(targetUsername)
	if err != nil || target == nil {
		return errors.New("Usuário não encontrado")
	}

	if strings.EqualFold(targetUsername, SuperuserUsername) && newRole != domain.RoleAdmin {
		return fmt.Errorf("não é permitido rebaixar a conta de superusuário ('%s')", SuperuserUsername)
	}

	if target.Role == domain.RoleAdmin && newRole != domain.RoleAdmin {
		admins, err := sm.userRepo.CountAdmins()
		if err != nil {
			return fmt.Errorf("falha ao verificar quantidade de administradores: %w", err)
		}
		if admins <= 1 {
			return errors.New("não é possível rebaixar o último administrador do sistema")
		}
	}

	if target.Role == newRole {
		return nil
	}

	if err := sm.userRepo.UpdateRole(targetUsername, newRole); err != nil {
		return fmt.Errorf("falha ao atualizar perfil: %w", err)
	}

	sm.revokeUserSessions(targetUsername)
	auditActor := strings.TrimSpace(actorUsername)
	if auditActor == "" {
		auditActor = targetUsername
	}
	sm.logAudit(auditActor, "USER_ROLE_UPDATED", fmt.Sprintf("Perfil de '%s' alterado de %s para %s", targetUsername, target.Role, newRole), clientIP)
	return nil
}

// UnlockAccount clears brute-force lockout counters for a username on administrator request.
func (sm *SecurityManager) UnlockAccount(actorUsername, targetUsername, clientIP string) error {
	targetUsername = strings.TrimSpace(targetUsername)
	if targetUsername == "" {
		return errors.New("Nome de usuário é obrigatório")
	}
	if sm.tracker != nil {
		sm.tracker.UnlockUser(targetUsername)
	}
	auditActor := strings.TrimSpace(actorUsername)
	if auditActor == "" {
		auditActor = SuperuserUsername
	}
	sm.logAudit(auditActor, "USER_UNLOCKED", fmt.Sprintf("Bloqueio de tentativas removido para '%s'", targetUsername), clientIP)
	return nil
}

// DeleteUser removes an operator or secondary administrator account.
// Only the designated master superuser (SuperuserUsername resolved from .env or config) is protected from deletion.
func (sm *SecurityManager) DeleteUser(username string) error {
	return sm.DeleteUserByActor(username, username, "")
}

// DeleteUserByActor removes an account, revokes active sessions, prevents deleting superuser/last admin, and records audit actor + IP.
func (sm *SecurityManager) DeleteUserByActor(actorUsername, username, clientIP string) error {
	username = strings.TrimSpace(username)
	user, err := sm.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		return errors.New("Usuário não encontrado")
	}

	if strings.EqualFold(username, SuperuserUsername) {
		return fmt.Errorf("não é permitido excluir a conta de superusuário do sistema ('%s')", SuperuserUsername)
	}

	if user.Role == domain.RoleAdmin {
		admins, err := sm.userRepo.CountAdmins()
		if err == nil && admins <= 1 {
			return errors.New("não é possível excluir o último administrador do sistema")
		}
	}

	if err := sm.userRepo.DeleteByUsername(username); err != nil {
		return err
	}
	sm.revokeUserSessions(username)
	if sm.tracker != nil {
		sm.tracker.UnlockUser(username)
	}

	auditActor := strings.TrimSpace(actorUsername)
	if auditActor == "" {
		auditActor = username
	}
	sm.logAudit(auditActor, "USER_DELETE", fmt.Sprintf("Conta de usuário '%s' (%s) excluída", username, user.Role), clientIP)
	return nil
}
