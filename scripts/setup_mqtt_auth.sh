#!/usr/bin/env bash
# Noxfort Monitor™ - MQTT Authentication Setup Script
# Generates and manages the Mosquitto password file with secure PBKDF2 hashing.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PASSWD_FILE="${ROOT_DIR}/mosquitto/config/passwd"
ENV_FILE="${ROOT_DIR}/.env"

# Extract existing MQTT credentials from environment or .env if present
DEFAULT_USER="${MQTT_USER:-}"
DEFAULT_PASS="${MQTT_PASSWORD:-}"

if [[ -z "${DEFAULT_USER}" && -f "${ENV_FILE}" ]]; then
    DEFAULT_USER=$(grep -E '^MQTT_USER=' "${ENV_FILE}" | cut -d '=' -f2- | tr -d ' "\r' || true)
fi

if [[ -z "${DEFAULT_PASS}" && -f "${ENV_FILE}" ]]; then
    DEFAULT_PASS=$(grep -E '^MQTT_PASSWORD=' "${ENV_FILE}" | cut -d '=' -f2- | tr -d ' "\r' || true)
fi

USER="${1:-${DEFAULT_USER:-noxfort_user}}"
PASS="${2:-${DEFAULT_PASS:-}}"

if [[ -z "${PASS}" ]]; then
    # Generate 16-character secure random password if none was specified
    if command -v openssl >/dev/null 2>&1; then
        PASS=$(openssl rand -hex 12)
    else
        PASS="NoxfortMqttSecure_$(date +%s)"
    fi
fi

if ! command -v mosquitto_passwd >/dev/null 2>&1; then
    echo "❌ Error: mosquitto_passwd is not installed. Install mosquitto or mosquitto-clients." >&2
    exit 1
fi

mkdir -p "$(dirname "${PASSWD_FILE}")"

# Create or update password file
if [[ -f "${PASSWD_FILE}" ]]; then
    echo "🔑 Updating user '${USER}' in existing password file..."
    mosquitto_passwd -b "${PASSWD_FILE}" "${USER}" "${PASS}"
else
    echo "🔑 Creating new Mosquitto password file with user '${USER}'..."
    mosquitto_passwd -c -b "${PASSWD_FILE}" "${USER}" "${PASS}"
fi

chmod 600 "${PASSWD_FILE}"
echo "✅ Mosquitto password file generated at: ${PASSWD_FILE}"

# Update or append to .env if .env exists
if [[ -f "${ENV_FILE}" ]]; then
    if grep -q '^MQTT_USER=' "${ENV_FILE}"; then
        sed -i "s|^MQTT_USER=.*|MQTT_USER=${USER}|" "${ENV_FILE}"
    else
        echo "MQTT_USER=${USER}" >> "${ENV_FILE}"
    fi

    if grep -q '^MQTT_PASSWORD=' "${ENV_FILE}"; then
        sed -i "s|^MQTT_PASSWORD=.*|MQTT_PASSWORD=${PASS}|" "${ENV_FILE}"
    else
        echo "MQTT_PASSWORD=${PASS}" >> "${ENV_FILE}"
    fi
    echo "✅ Synchronized MQTT credentials into .env"
fi

echo "=========================================="
echo " MQTT Credentials Configured Successfully"
echo " Username: ${USER}"
echo " Password: ${PASS}"
echo "=========================================="
