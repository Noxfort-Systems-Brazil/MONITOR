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
// File: web/static/js/settings_db_api.js
// Author: Gabriel Moraes
// Date: October 2026

class DatabaseApiClient {
    async getStatus() {
        const res = await fetch('/api/settings/database/status');
        if (!res.ok) {
            throw new Error(`HTTP ${res.status}: Failed to fetch database status`);
        }
        return await res.json();
    }

    async testConnection(payload) {
        const res = await fetch('/api/settings/database/test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        return { ok: res.ok, data };
    }

    async saveConnection(payload) {
        const res = await fetch('/api/settings/database/save', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        return { ok: res.ok, data };
    }

    async provisionUser(payload) {
        const res = await fetch('/api/settings/database/provision-user', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        return { ok: res.ok, data };
    }
}

