#!/usr/bin/env bash
# Noxfort Monitor™ - Database Backup CLI Script
# Supports both standalone SQLite and PostgreSQL setups.

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
