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
// File: internal/transport/http/database_handler.go
// Author: Gabriel Moraes
// Date: October 2026

package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"noxfort-monitor-server/internal/domain"
)

// DatabaseService defines the operations required by DatabaseHandler to monitor, test, and switch storage engines.
type DatabaseService interface {
	GetStatus() domain.DatabaseStatus
	GetConfig() domain.DatabaseConfig
	TestConnection(cfg domain.DatabaseConfig) (domain.DatabaseStatus, error)
	Switch(newCfg domain.DatabaseConfig, migrate bool) error
}

// BackupService defines operations for creating and retrieving database backups.
type BackupService interface {
	CreateBackup() (domain.BackupMetadata, error)
	ListBackups() ([]domain.BackupMetadata, error)
}

// BackupRestoreService extends BackupService with validated path resolution and restore execution.
type BackupRestoreService interface {
	ResolveBackupPath(filename string) (string, error)
	RestoreBackup(filename string) error
}

// DatabaseHandler manages database connection settings, testing, and schema provisioning APIs.
type DatabaseHandler struct {
	dbService     DatabaseService
	auditRepo     domain.AuditRepository
	provisioner   DatabaseUserProvisioner
	backupService BackupService
}

// NewDatabaseHandler initializes the DatabaseHandler with the default PostgreSQL provisioner.
func NewDatabaseHandler(dbService DatabaseService, auditRepo domain.AuditRepository) *DatabaseHandler {
	return NewDatabaseHandlerWithProvisioner(dbService, auditRepo, &PostgresUserProvisioner{})
}

// NewDatabaseHandlerWithProvisioner initializes the DatabaseHandler with a custom user provisioner for testability.
func NewDatabaseHandlerWithProvisioner(dbService DatabaseService, auditRepo domain.AuditRepository, provisioner DatabaseUserProvisioner) *DatabaseHandler {
	return &DatabaseHandler{
		dbService:   dbService,
		auditRepo:   auditRepo,
		provisioner: provisioner,
	}
}

// SetBackupService attaches a BackupService to handle database backup requests.
func (h *DatabaseHandler) SetBackupService(bs BackupService) {
	h.backupService = bs
}

// ServePage renders the dedicated Server & Database configuration page.
func (h *DatabaseHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Server & Storage Engine",
	}

	if err := RenderPageWithRequest(w, r, "web/templates/server.html", data); err != nil {
		log.Printf("[SERVER_PAGE] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}
}

// HandleStatus returns the live status and active configuration of the database.
func (h *DatabaseHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.dbService.GetStatus()
	cfg := h.dbService.GetConfig()

	resp := map[string]interface{}{
		"status": status,
		"config": map[string]interface{}{
			"type":      cfg.Type,
			"host":      cfg.Host,
			"port":      cfg.Port,
			"user":      cfg.User,
			"dbname":    cfg.DBName,
			"schema":    cfg.Schema,
			"sslmode":   cfg.SSLMode,
			"file_path": cfg.FilePath,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleTest evaluates connectivity against supplied PostgreSQL or SQLite credentials.
func (h *DatabaseHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, _ := parseDatabaseConfig(r)
	status, err := h.dbService.TestConnection(cfg)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"status":  status,
		})
		return
	}

	message := formatConnectionTestMessage(status, cfg)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
		"status":  status,
	})
}

// HandleSave applies new database settings, creates the schema if needed, and hot-reloads the connection.
func (h *DatabaseHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, migrate := parseDatabaseConfig(r)

	if err := h.dbService.Switch(cfg, migrate); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao inicializar banco: %v", err),
		})
		return
	}

	details := fmt.Sprintf("Motor: %s, Banco: %s, Schema: %s, Migração: %v", cfg.Type, cfg.DBName, cfg.Schema, migrate)
	logSecurityAuditEvent(h.auditRepo, r, "DATABASE_CONFIG_UPDATED", details)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Banco de dados configurado com sucesso no schema '%s'!", cfg.Schema),
		"status":  h.dbService.GetStatus(),
	})
}

// HandleBackup creates a database backup snapshot.
func (h *DatabaseHandler) HandleBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if h.backupService == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Serviço de backup não configurado",
		})
		return
	}

	info, err := h.backupService.CreateBackup()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao gerar backup: %v", err),
		})
		return
	}

	logSecurityAuditEvent(h.auditRepo, r, "DATABASE_BACKUP_CREATED", fmt.Sprintf("Arquivo: %s, Tamanho: %d bytes, Motor: %s", info.Filename, info.SizeBytes, info.Driver))

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Backup gerado com sucesso!",
		"backup":  info,
	})
}

// HandleListBackups returns all existing database backup files.
func (h *DatabaseHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.backupService == nil {
		_ = json.NewEncoder(w).Encode([]domain.BackupMetadata{})
		return
	}

	list, err := h.backupService.ListBackups()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao listar backups: %v", err),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(list)
}

// HandleDownloadBackup securely streams an existing backup file to an authenticated administrator.
func (h *DatabaseHandler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resolver, ok := h.backupService.(BackupRestoreService)
	if !ok || resolver == nil {
		http.Error(w, `{"success":false,"error":"Serviço de backup indisponível"}`, http.StatusNotImplemented)
		return
	}

	filename := strings.TrimSpace(r.URL.Query().Get("file"))
	fullPath, err := resolver.ResolveBackupPath(filename)
	if err != nil {
		http.Error(w, `{"success":false,"error":"Arquivo de backup inválido ou não encontrado"}`, http.StatusBadRequest)
		return
	}

	logSecurityAuditEvent(h.auditRepo, r, "DATABASE_BACKUP_DOWNLOADED", fmt.Sprintf("Backup baixado: %s", filepath.Base(fullPath)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(fullPath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fullPath)
}

// HandleRestoreBackup restores the active database from a previously generated snapshot.
func (h *DatabaseHandler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	restorer, ok := h.backupService.(BackupRestoreService)
	if !ok || restorer == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Restauração de backup não configurada",
		})
		return
	}

	filename := strings.TrimSpace(r.FormValue("filename"))
	if filename == "" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req struct {
			Filename string `json:"filename"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			filename = strings.TrimSpace(req.Filename)
		}
	}

	if filename == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Nome do arquivo de backup é obrigatório",
		})
		return
	}

	if err := restorer.RestoreBackup(filename); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha ao restaurar backup: %v", err),
		})
		return
	}

	logSecurityAuditEvent(h.auditRepo, r, "DATABASE_BACKUP_RESTORED", fmt.Sprintf("Banco restaurado a partir de: %s", filename))
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Banco restaurado com sucesso a partir de %s!", filename),
	})
}
