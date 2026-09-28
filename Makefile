# Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
# Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# File: Makefile
# Author: Gabriel Moraes
# Date: 2026-01-13

# Binary output name
BINARY_NAME=bin/noxfort-monitor
MAIN_PATH=cmd/server/main.go
MOSQUITTO_CONF=mosquitto/config/mosquitto.conf

# Default target (what runs when you just type 'make')
all: build

# 1. Build the executable
build:
	@echo "🔨 Building Noxfort Monitor Desktop (Wails v2)..."
	@mkdir -p bin
	GOTOOLCHAIN=local go build -tags "production,webkit2_41" -ldflags="-s -w" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build successful! Desktop Binary created at $(BINARY_NAME)"

# 2. Run the application directly (Development mode)
#    Make sure to run 'make broker-start' first.
run:
	@echo "🚀 Running Desktop Application... (ensure broker is running: make broker-start)"
	GOTOOLCHAIN=local go run -tags "production,webkit2_41" $(MAIN_PATH)

# 2b. Run in headless mode without GUI
run-headless:
	@echo "🚀 Running Server in Headless Mode... (ensure broker is running: make broker-start)"
	GOTOOLCHAIN=local go run -tags "production,webkit2_41" $(MAIN_PATH) --headless

# 3. Clean up build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f monitor_logs.db
	@echo "✨ Clean complete."

# 4. Lint and Static Analysis
lint:
	@echo "🔍 Running static analysis (go vet)..."
	go vet ./...

# 5. Run automated tests (Backend + Frontend)
test: test-backend test-frontend

test-backend:
	@echo "🧪 Running Backend Tests..."
	go test ./... -v

test-frontend:
	@echo "🧪 Running Frontend Tests..."
	npm test

# 5. Install/Update Dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod tidy
	go mod download

# 6. Cross-compile for Linux
build-linux:
	@echo "🐧 Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux $(MAIN_PATH)
	@echo "✅ Linux binary ready."

# ---- MQTT Broker (Mosquitto - native, no Docker required) ----

# 7. Start the Mosquitto MQTT broker
#    Uses systemctl if available (recommended), otherwise runs directly.
broker-start:
	@echo "🟢 Starting Mosquitto MQTT broker..."
	@mkdir -p mosquitto/data mosquitto/log
	@if systemctl is-active --quiet mosquitto 2>/dev/null; then \
		echo "✅ Mosquitto is already running via systemd."; \
	elif command -v systemctl >/dev/null 2>&1; then \
		sudo systemctl start mosquitto && echo "✅ Broker started via systemd."; \
	else \
		mosquitto -c $(MOSQUITTO_CONF) -d && echo "✅ Broker started in background."; \
	fi

# 8. Stop the Mosquitto MQTT broker
broker-stop:
	@echo "🔴 Stopping Mosquitto MQTT broker..."
	@if command -v systemctl >/dev/null 2>&1; then \
		sudo systemctl stop mosquitto && echo "✅ Broker stopped."; \
	else \
		pkill -x mosquitto && echo "✅ Broker stopped." || echo "⚠️  Mosquitto was not running."; \
	fi

# 9. Check if the Mosquitto broker is running
broker-status:
	@echo "📡 MQTT Broker status:"
	@if systemctl is-active --quiet mosquitto 2>/dev/null; then \
		echo "  ✅ Mosquitto is running (systemd service)"; \
	elif pgrep -x mosquitto > /dev/null; then \
		echo "  ✅ Mosquitto is running (standalone process, PID: $$(pgrep -x mosquitto))"; \
	else \
		echo "  ❌ Mosquitto is NOT running. Start it with: make broker-start"; \
	fi

# 10. Install Mosquitto (if not already installed)
broker-install:
	@echo "📦 Installing Mosquitto..."
	sudo apt-get update -qq && sudo apt-get install -y mosquitto
	@echo "✅ Mosquitto installed."

# 10b. Configure Mosquitto Password and Authentication
broker-auth:
	@chmod +x scripts/setup_mqtt_auth.sh
	@./scripts/setup_mqtt_auth.sh

# 10c. Create and manage Database Backups
backup:
	@chmod +x scripts/backup.sh
	@./scripts/backup.sh

backup-list:
	@echo "📦 Existing Database Backups:"
	@mkdir -p backups && ls -lh backups/

# 11. Generate .deb Installer Package
deb:
	@chmod +x build_installer.sh
	@./build_installer.sh

# ---- Caddy & Edge Reverse Proxy (Docker) ----

DOCKER_COMPOSE := $(shell if command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; elif docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo ""; fi)

# 12. Build Caddy image with DuckDNS plugin
caddy-build:
	@echo "🔨 Building Caddy image with DuckDNS DNS-01 plugin..."
	@if [ -n "$(DOCKER_COMPOSE)" ]; then \
		$(DOCKER_COMPOSE) build caddy; \
	else \
		docker build -t noxfort-caddy ./caddy; \
	fi
	@echo "✅ Caddy build complete."

# 13. Start edge services (Mosquitto + Caddy)
services-start:
	@echo "🟢 Starting edge services (Mosquitto + Caddy)..."
	@if [ -n "$(DOCKER_COMPOSE)" ]; then \
		$(DOCKER_COMPOSE) up -d; \
		echo "✅ Services started. Caddy listening on ports 80/443."; \
	else \
		echo "⚠️  Docker Compose não encontrado. Instale com: sudo apt install docker-compose-v2"; \
		exit 1; \
	fi

# 14. Stop edge services
services-stop:
	@echo "🔴 Stopping edge services..."
	@if [ -n "$(DOCKER_COMPOSE)" ]; then \
		$(DOCKER_COMPOSE) down; \
		echo "✅ Services stopped."; \
	else \
		echo "⚠️  Docker Compose não encontrado."; \
	fi

# 15. View Caddy and Mosquitto live logs
services-logs:
	@if [ -n "$(DOCKER_COMPOSE)" ]; then \
		$(DOCKER_COMPOSE) logs -f caddy; \
	else \
		echo "⚠️  Docker Compose não encontrado."; \
	fi

.PHONY: all build run run-headless clean lint test test-backend test-frontend deps build-linux broker-start broker-stop broker-status broker-install broker-auth backup backup-list deb caddy-build services-start services-stop services-logs