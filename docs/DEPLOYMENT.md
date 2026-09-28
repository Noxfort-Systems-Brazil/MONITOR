[📚 Documentation Hub](INDEX.md) > **Production Deployment Guide**

---

# 🚀 Production Deployment Guide: Noxfort Monitor™

This document provides step-by-step instructions for deploying **Noxfort Monitor™ v2.0** in production industrial and enterprise environments, covering standalone **Systemd** service setup in **Headless** mode, streamlined **Debian package (`.deb`)** installation, **PostgreSQL** database configuration, and **NGINX** reverse proxy deployment with **SSL**.

---

## 1. Production Deployment Options

You can deploy Noxfort Monitor in two primary ways:
1. **Native `.deb` Package (Recommended for Ubuntu/Debian workstations)**: Automatically installs the binary, dependencies, Mosquitto service, desktop integration, and system shortcuts.
2. **Dedicated Systemd Service in Headless Mode (Recommended for headless servers and cloud VPS instances)**.

---

## 2. Installation via Debian Package (`.deb`)

On your build machine or CI/CD runner:
```bash
# Generates the package in build_deb/noxfort-monitor_2.0.1_amd64.deb
make deb
```

On the target host:
```bash
sudo dpkg -i noxfort-monitor_2.0.1_amd64.deb
sudo apt-get install -f # Resolves missing dependencies if needed
```

This installs the software to `/opt/noxfort-monitor/`, symlinks `/usr/local/bin/noxfort-monitor`, installs desktop application icons, and enables Mosquitto MQTT via systemd.

---

## 3. Linux Service Configuration (Headless Systemd)

For dedicated servers without a display server (X11 or Wayland), the binary **must be launched with the `--headless` flag**.

### 3.1 Build the Production Binary
```bash
make build-linux
# Binary compiled to bin/noxfort-monitor-linux
```

Copy the binary to the server:
```bash
scp bin/noxfort-monitor-linux user@production-server:/usr/local/bin/noxfort-monitor
```

### 3.2 Create Dedicated System User and Permissions
```bash
sudo useradd -m -s /bin/bash noxfort
sudo chown noxfort:noxfort /usr/local/bin/noxfort-monitor
sudo chmod +x /usr/local/bin/noxfort-monitor
```

### 3.3 Create the Systemd Unit File
Create `/etc/systemd/system/noxfort-monitor.service`:

```ini
[Unit]
Description=Noxfort Monitor Server (Headless Production Service)
After=network.target mosquitto.service postgresql.service
Wants=mosquitto.service

[Service]
Type=simple
User=noxfort
Group=noxfort

# Headless flag is required on servers without a graphical display
ExecStart=/usr/local/bin/noxfort-monitor --headless

Restart=on-failure
RestartSec=5

# Working directory containing production .env
WorkingDirectory=/home/noxfort

# Environment variables
Environment="PORT=22100"
EnvironmentFile=-/home/noxfort/.env

# Resource limits (optional)
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

### 3.4 Configure the Production `.env` File
Create `/home/noxfort/.env`:
```ini
MONITOR_ADMIN_USER=corporate_admin
MONITOR_ADMIN_PASSWORD=StrongEncryptedPassword123!
MQTT_USER=noxfort_user
MQTT_PASSWORD=SuperSecureMqttPass2026!
PORT=22100
```
Restrict file permissions (mandatory for security):
```bash
sudo chmod 600 /home/noxfort/.env
sudo chown noxfort:noxfort /home/noxfort/.env
```

### 3.5 Enable and Start the Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable noxfort-monitor
sudo systemctl start noxfort-monitor
```

Inspect service health and real-time logs:
```bash
sudo systemctl status noxfort-monitor
sudo journalctl -u noxfort-monitor -f
```

---

## 4. PostgreSQL Configuration in Production

For high-concurrency environments and regulatory compliance:

1. **Install PostgreSQL 14+**:
   ```bash
   sudo apt-get install -y postgresql postgresql-contrib
   ```
2. **Create Database and User with Schema Access**:
   The Monitor provisions its own tables automatically. Ensure the target database exists:
   ```sql
   CREATE DATABASE noxfort_database;
   CREATE USER user_monitor WITH PASSWORD 'your_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE noxfort_database TO user_monitor;
   ```
3. **Connect via Desktop Application**:
   Open the native desktop window, access the Database settings tab (`/server`) or invoke the API endpoint `/api/settings/database/save` targeting PostgreSQL. The system automatically creates `schema_monitor`, tables, and indices. See [Database & Dual-Engine Persistence](DATABASE.md).

---

## 5. Edge Reverse Proxy & SSL Termination (Caddy or NGINX)

To expose telemetry ingestion securely over the public internet, a reverse proxy provides TLS/SSL termination and DDoS mitigation:

### 5.1 Option A: Caddy Server with DuckDNS (Recommended)
Noxfort Monitor provides a turnkey Docker-based Caddy edge gateway with automated **DNS-01 ACME challenges** via DuckDNS, eliminating manual certificate renewal and opening ports 80/443 securely. See [Caddy Reverse Proxy & DuckDNS](CADDY_INTEGRATION.md).

```bash
# Build Caddy with DuckDNS plugin and start Mosquitto + Caddy
make caddy-build
make services-start
```

### 5.2 Option B: Custom NGINX Reverse Proxy
If deploying on a server with an existing NGINX instance, configure SSL termination for the telemetry ingestion endpoint:

```nginx
server {
    listen 80;
    server_name monitor.yourcompany.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitor.yourcompany.com;

    ssl_certificate /etc/letsencrypt/live/monitor.yourcompany.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.yourcompany.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Telemetry ingestion endpoint for external IoT edge nodes
    location /api/telemetry {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }

    # Browser requests to root return 403 Forbidden by design
    location / {
        proxy_pass http://127.0.0.1:22100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

> [!NOTE]
> Direct web browser access to HTML dashboard routes returns **403 Forbidden**. Operator interactions are performed exclusively via the native desktop application.

---

## 6. Remote Access: DuckDNS vs Ngrok Tunnel

When the server operates behind industrial firewalls or carrier-grade NAT (CGNAT) without public static IPs:
* **DuckDNS + Caddy (Dynamic DNS)**: The recommended architecture for public access with port forwarding or native IPv6. See [Caddy Integration](CADDY_INTEGRATION.md).
* **Ngrok Reverse Tunnel (Zero-Port-Forwarding)**: When inbound ports cannot be opened, Ngrok creates an encrypted outbound tunnel to receive telemetry from remote agents (Carina, Synapse). See [Remote Access Guide](REMOTE_ACCESS.md).

---

## 7. Automated Backups in Production

Configure an automated daily backup job using cron:

```bash
# Edit the noxfort user's crontab
sudo crontab -u noxfort -e

# Add a daily backup scheduled at 03:00 AM with automatic 7-day retention:
0 3 * * * /opt/noxfort-monitor/scripts/backup.sh >> /var/log/noxfort-backup.log 2>&1
```

---

## 8. Production Observability & Metrics

For enterprise monitoring platforms (e.g., Prometheus, Grafana, Datadog):
1. **Liveness Probe**: Configure container/load balancer health checks pointing to `http://<server-ip>:22100/healthz`.
2. **Prometheus Scraping Job**: Add the following target to `/etc/prometheus/prometheus.yml`:
   ```yaml
   scrape_configs:
     - job_name: 'noxfort-monitor'
       scrape_interval: 15s
       metrics_path: '/metrics'
       static_configs:
         - targets: ['127.0.0.1:22100']
   ```

---

### 🔗 Related Documentation
* 🖥️ [Desktop Application](DESKTOP_APP.md) — Local operations with Wails GUI
* 🗄️ [Database & Persistence](DATABASE.md) — Configuration and migration to PostgreSQL
* 🔐 [Security & RBAC](SECURITY.md) — Account configuration and superuser bootstrapping
* 🌐 [Remote Access](REMOTE_ACCESS.md) — Secure tunneling for external edge nodes
* 📡 [API Reference](API_REFERENCE.md) — Monitoring endpoints specification
* 🧪 [Testing Guide](TESTING.md) — Post-deployment verification
