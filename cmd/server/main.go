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
// File: cmd/server/main.go
// Author: Gabriel Moraes
// Date: October 2026

package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	// Internal Packages
	"noxfort-monitor-server/internal/desktop"
	"noxfort-monitor-server/internal/monitor"
	"noxfort-monitor-server/internal/security"
	"noxfort-monitor-server/internal/storage"
	transportHttp "noxfort-monitor-server/internal/transport/http"
	transportMqtt "noxfort-monitor-server/internal/transport/mqtt"
	"noxfort-monitor-server/internal/tunnel"
)

func main() {
	headless := flag.Bool("headless", false, "Run in headless mode without GUI desktop window")
	flag.BoolVar(headless, "server-only", false, "Alias for --headless")
	demo := flag.Bool("demo", false, "Run in demo/presentation mode (serves desktop UI on HTTP for automated recording)")
	flag.Parse()

	// 0. Single Instance Lock (Atomic OS flock + IPC window activation)
	var singleInstanceServer io.Closer
	var appRef *desktop.App
	var appRefMu sync.Mutex

	lock, err := desktop.AcquireLockOrActivate(func() {
		appRefMu.Lock()
		app := appRef
		appRefMu.Unlock()
		if app != nil {
			app.RestoreWindow()
		}
	})
	if errors.Is(err, desktop.ErrAlreadyRunning) {
		log.Println("[BOOT] Uma instância do Noxfort Monitor já está em execução. Janela existente trazida para o primeiro plano.")
		os.Exit(0)
	} else if err != nil {
		log.Printf("[WARN] Aviso ao verificar instância única: %v", err)
	} else {
		singleInstanceServer = lock
	}

	// 1. Initialize Logger, Load .env & Audit Environment Security
	log.Println("[BOOT] Starting Noxfort Monitor v2.0 (Event-Driven)...")
	security.LoadDotEnv()
	security.ValidateEnvironmentSecurity(".env")

	// 2. Database Connection (PostgreSQL with auto-schema or SQLite fallback)
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[FATAL] Could not resolve home directory: %v", err)
	}
	dataDir := filepath.Join(homedir, "Documentos", "Monitor")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("[FATAL] Could not create data directory %s: %v", dataDir, err)
	}
	dbPath := filepath.Join(dataDir, "monitor_logs.db")
	log.Printf("[BOOT] Data directory: %s", dataDir)

	dbConfig := storage.LoadDatabaseConfig()
	if dbConfig.FilePath == "" {
		dbConfig.FilePath = dbPath
	}

	var db *sql.DB
	var activeDriver string

	if dbConfig.Type == "postgres" {
		log.Printf("[BOOT] Connecting to PostgreSQL at %s:%d (Database: %s, Schema: %s)...",
			dbConfig.Host, dbConfig.Port, dbConfig.DBName, dbConfig.Schema)
		pgDB, drv, pgErr := storage.OpenConnection(dbConfig)
		if pgErr == nil {
			if initErr := storage.InitPostgresSchema(pgDB, dbConfig.Schema); initErr == nil {
				db = pgDB
				activeDriver = drv
				log.Printf("[INFO] PostgreSQL connected successfully (Schema: '%s').", dbConfig.Schema)
			} else {
				log.Printf("[WARN] PostgreSQL connected but failed to initialize schema '%s': %v. Falling back to SQLite.", dbConfig.Schema, initErr)
				_ = pgDB.Close()
			}
		} else {
			log.Printf("[WARN] Failed to connect to PostgreSQL (%v). Falling back to local SQLite.", pgErr)
		}
	}

	// Fallback to SQLite if PostgreSQL was not chosen or failed
	if db == nil {
		sqliteDB, err := storage.NewDatabase(dbPath)
		if err != nil {
			log.Fatalf("[FATAL] Failed to initialize local SQLite database: %v", err)
		}
		db = sqliteDB
		activeDriver = "sqlite"
		dbConfig.Type = "sqlite"
		log.Println("[INFO] SQLite connected (Pure Go Driver).")
	}

	// 2.1 Central DB Manager (supports live hot-reload from Web/Desktop UI)
	dbManager := storage.NewDBManager(db, activeDriver, dbConfig)

	// 3. Initialize Repositories
	deviceRepo := storage.NewDeviceRepository(db)
	contactRepo := storage.NewContactRepository(db)
	settingsRepo := storage.NewSettingsRepository(db)
	telemetryRepo := storage.NewTelemetryRepository(db)
	userRepo := storage.NewUserRepository(db)
	auditRepo := storage.NewAuditRepository(db, activeDriver)
	sessionStore := storage.NewDBSessionStore(db, activeDriver, security.SessionTTL)

	// Register repositories for live database hot-reloading
	dbManager.RegisterRepository(deviceRepo)
	dbManager.RegisterRepository(contactRepo)
	dbManager.RegisterRepository(settingsRepo)
	dbManager.RegisterRepository(telemetryRepo)
	dbManager.RegisterRepository(userRepo)
	dbManager.RegisterRepository(auditRepo)
	dbManager.RegisterRepository(sessionStore)

	// 4. Initialize Core & Security Services
	emailChan := monitor.NewEmailChannel()
	telegramChan := monitor.NewTelegramChannel()
	channelTester := monitor.NewChannelTester(emailChan, telegramChan)

	alertService := monitor.NewAlertService(contactRepo, settingsRepo, emailChan, telegramChan)
	alertService.SetAuditRepository(auditRepo)

	// High-throughput buffered batch telemetry writer (shields DB from I/O storms)
	batchWriter := storage.NewBufferedTelemetryWriter(telemetryRepo, 50, 100*time.Millisecond)
	stateManager := monitor.NewStateManager(batchWriter, deviceRepo, alertService)
	secManager := security.NewPersistentSecurityManager(userRepo, sessionStore)
	secManager.SetAuditRepository(auditRepo)

	if err := secManager.EnsureSuperuser(); err != nil {
		log.Fatalf("[FATAL] Failed to initialize superuser: %v", err)
	}

	engine := monitor.NewEngine(deviceRepo, telemetryRepo, alertService)
	engine.SetAuditRepository(auditRepo)
	engine.Start()

	// 5. Initialize & Start MQTT Client
	settings, err := settingsRepo.GetSettings()
	if err != nil {
		log.Printf("[WARN] Failed to load settings for MQTT (using defaults): %v", err)
	}
	brokerURL := "tcp://127.0.0.1:1883"
	if settings.MqttAddress != "" {
		brokerURL = settings.MqttAddress
	}
	log.Printf("[BOOT] Connecting to MQTT Broker at %s...", brokerURL)
	mqttClient := transportMqtt.NewClient(brokerURL, stateManager)
	strictMQTT := !strings.EqualFold(strings.TrimSpace(os.Getenv("MONITOR_MQTT_STRICT")), "false")
	mqttClient.SetDevicePolicy(deviceRepo, strictMQTT)
	log.Printf("[BOOT] MQTT Zero-Trust Policy: strict=%v", strictMQTT)
	bgCtx, bgCancel := context.WithCancel(context.Background())
	sessionStore.StartCleanup(bgCtx, 1*time.Hour)
	if err := mqttClient.Connect(); err != nil {
		log.Printf("[WARN] Initial MQTT broker connection failed (%v). Retrying in background so UI and HTTP ingestion stay available...", err)
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-bgCtx.Done():
					return
				case <-ticker.C:
					if mqttClient.IsConnected() {
						return
					}
					if err := mqttClient.Connect(); err == nil {
						log.Println("[INFO] MQTT Listener Active (Reconnected in background).")
						return
					}
				}
			}
		}()
	} else {
		log.Println("[INFO] MQTT Listener Active (Listening for JSON events).")
	}

	// 6. Initialize HTTP Server & Remote Access Manager (DuckDNS)
	port := os.Getenv("PORT")
	if port == "" {
		port = "22100"
	}

	tunnelDriver := tunnel.NewDuckDNSDriver()
	tunnelManager := tunnel.NewManager(tunnelDriver, port)

	activeToken := settings.DuckDNSToken
	activeDomain := settings.DuckDNSDomain
	activeEnabled := settings.DuckDNSEnabled

	if activeToken != "" && activeDomain != "" && activeEnabled && !strings.Contains(activeDomain, "ngrok") {
		log.Printf("[BOOT] Auto-starting DuckDNS updater on domain '%s'...", activeDomain)
		if err := tunnelManager.Start(activeToken, activeDomain); err != nil {
			log.Printf("[WARN] Failed to auto-start DuckDNS updater on boot: %v", err)
		}
	}

	backupDir := filepath.Join(dataDir, "backups")
	backupManager := storage.NewBackupManager(dbManager, backupDir, 7)

	retentionDays := 90
	if raw := strings.TrimSpace(os.Getenv("MONITOR_RETENTION_DAYS")); raw != "" {
		if parsed, convErr := strconv.Atoi(raw); convErr == nil {
			retentionDays = parsed
		}
	}
	storage.NewRetentionManager(dbManager, retentionDays).Start(bgCtx, 24*time.Hour)

	backupIntervalHours := 24
	if raw := strings.TrimSpace(os.Getenv("MONITOR_BACKUP_INTERVAL_HOURS")); raw != "" {
		if parsed, convErr := strconv.Atoi(raw); convErr == nil {
			backupIntervalHours = parsed
		}
	}
	if backupIntervalHours > 0 {
		backupManager.StartScheduledBackups(bgCtx, time.Duration(backupIntervalHours)*time.Hour)
	}

	httpServer := transportHttp.NewServer(
		":"+port,
		deviceRepo,
		telemetryRepo,
		contactRepo,
		settingsRepo,
		stateManager,
		channelTester,
		secManager,
		tunnelManager,
		dbManager,
		auditRepo,
	)
	httpServer.SetMQTTStatus(mqttClient)
	httpServer.SetBackupService(backupManager)
	httpServer.SetOnMQTTConfigChanged(func(newAddr string) {
		log.Printf("[SETTINGS] Reconnecting MQTT broker to %s...", newAddr)
		if err := mqttClient.Reconnect(newAddr); err != nil {
			log.Printf("[WARN] MQTT reconnect to %s failed: %v", newAddr, err)
		}
	})

	localIP := transportHttp.GetLocalIP()
	serverURL := fmt.Sprintf("http://%s:%s", localIP, port)

	// Run HTTP server in background
	go func() {
		if *demo {
			log.Printf("[INFO] Demo UI Server running at %s (Full Desktop UI exposed for automated recording)", serverURL)
			if err := httpServer.RunDesktopServer(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("[FATAL] Web Server failed: %v", err)
			}
		} else {
			log.Printf("[INFO] Ingestion Telemetry Server running at %s (External endpoint: POST /api/telemetry | Browser UI access disabled)", serverURL)
			if err := httpServer.Run(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("[FATAL] Web Server failed: %v", err)
			}
		}
	}()

	// 7. Graceful Shutdown Coordinator (guaranteed to execute at most once)
	var shutdownOnce sync.Once
	shutdown := func() {
			shutdownOnce.Do(func() {
			log.Println("[SHUTDOWN] Stopping Noxfort Monitor...")
			bgCancel()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if singleInstanceServer != nil {
				_ = singleInstanceServer.Close()
			}
			if err := tunnelManager.Stop(); err != nil {
				log.Printf("[WARN] Tunnel shutdown error: %v", err)
			}
			if err := httpServer.Stop(shutdownCtx); err != nil {
				log.Printf("[WARN] HTTP Server shutdown error: %v", err)
			}
			engine.Stop()
			mqttClient.Disconnect()
			batchWriter.Close()
			if err := alertService.Shutdown(shutdownCtx); err != nil {
				log.Printf("[WARN] Timed out waiting for pending alerts to finish: %v", err)
			}
			if activeDB := dbManager.GetDB(); activeDB != nil {
				_ = activeDB.Close()
			}
			log.Println("[SHUTDOWN] All services stopped cleanly.")
		})
	}
	defer shutdown()

	// 8. Run in Headless, Demo or Desktop Mode
	if *headless || *demo {
		if *demo {
			log.Println("[BOOT] Running in DEMO mode (UI exposed on HTTP for automated recording). Press Ctrl+C to stop.")
		} else {
			log.Println("[BOOT] Running in HEADLESS mode (no GUI window). Press Ctrl+C to stop.")
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		shutdown()
		os.Exit(0)
	} else {
		log.Println("[BOOT] Starting Noxfort Monitor Desktop GUI (Wails v2)...")
		desktopApp := desktop.New(httpServer.DesktopHandler(), shutdown)
		appRefMu.Lock()
		appRef = desktopApp
		appRefMu.Unlock()

		if err := desktopApp.Run(); err != nil {
			log.Fatalf("[FATAL] Desktop application failed: %v", err)
		}
		shutdown()
		os.Exit(0)
	}
}
