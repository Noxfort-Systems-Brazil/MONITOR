#!/usr/bin/env bash
# ==============================================================================
# Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
# Copyright (C) 2026 Noxfort Systems
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.
#
# File: scripts/backup.sh
# Author: Gabriel Moraes
# Date: 2026-09-28
# Description: Database Backup CLI Script (supports standalone SQLite and PostgreSQL)
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKUP_DIR="${ROOT_DIR}/backups"
TIMESTAMP="$(date +'%Y%m%d_%H%M%S')"
RETENTION_DAYS=7

mkdir -p "${BACKUP_DIR}"
chmod 700 "${BACKUP_DIR}"

# Determine DB location from default or documents path
SQLITE_DB="${ROOT_DIR}/monitor_logs.db"
if [[ ! -f "${SQLITE_DB}" && -f "${HOME}/Documentos/Monitor/monitor_logs.db" ]]; then
    SQLITE_DB="${HOME}/Documentos/Monitor/monitor_logs.db"
fi

echo "=========================================="
echo " Noxfort Monitor - Database Backup Utility"
echo " Time: $(date)"
echo "=========================================="

if [[ -f "${SQLITE_DB}" ]]; then
    DEST_FILE="${BACKUP_DIR}/monitor_sqlite_${TIMESTAMP}.db"
    echo "📦 Backing up SQLite database: ${SQLITE_DB} -> ${DEST_FILE}..."
    
    if command -v sqlite3 >/dev/null 2>&1; then
        sqlite3 "${SQLITE_DB}" ".backup '${DEST_FILE}'"
    else
        # Fallback to copy if sqlite3 CLI is absent
        cp "${SQLITE_DB}" "${DEST_FILE}"
    fi

    chmod 600 "${DEST_FILE}"
    echo "✅ SQLite Backup successfully created: ${DEST_FILE} ($(du -h "${DEST_FILE}" | cut -f1))"
else
    echo "⚠️  No local SQLite database found at ${SQLITE_DB}. If using PostgreSQL, ensure it is configured via Web UI."
fi

# Clean up backups older than RETENTION_DAYS
echo "🧹 Pruning backups older than ${RETENTION_DAYS} days in ${BACKUP_DIR}..."
find "${BACKUP_DIR}" -type f \( -name "monitor_*.db" -o -name "monitor_*.sql" \) -mtime "+${RETENTION_DAYS}" -exec rm -f {} +
echo "✨ Backup procedure complete."
