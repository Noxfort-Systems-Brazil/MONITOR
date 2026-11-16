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
// File: web/static/js/remote_api.js
// Author: Gabriel Moraes
// Date: October 2026

const RemoteTunnelAPI = {
    /**
     * Safely parses JSON response payloads and falls back to plain text on unexpected errors.
     */
    async _parseResponseData(res) {
        try {
            return await res.json();
        } catch (_) {
            const text = typeof res.text === 'function' ? await res.text().catch(() => '') : '';
            const msg = text || `HTTP ${res.status || 500}`;
            return { success: false, error: msg, message: msg };
        }
    },

    /**
     * Retrieves current tunnel state, URLs, and IPv6 information.
     */
    async getStatus() {
        try {
            const res = await fetch('/api/tunnel/status');
            if (res.ok) {
                return await res.json();
            }
        } catch (e) {
            console.error('Error fetching tunnel status:', e);
        }
        return null;
    },

    /**
     * Persists DuckDNS credentials without starting the service.
     */
    async saveConfig(token, domain) {
        const body = new URLSearchParams();
        body.append('duckdns_token', token);
        body.append('duckdns_domain', domain);

        const res = await fetch('/api/tunnel/save', {
            method: 'POST',
            body: body,
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Accept': 'application/json'
            }
        });
        const data = await this._parseResponseData(res);
        return { ok: res.ok && data.success, data };
    },

    /**
     * Starts the DuckDNS periodic updater and remote access endpoint.
     */
    async startTunnel(token, domain) {
        const body = new URLSearchParams();
        body.append('duckdns_token', token);
        body.append('duckdns_domain', domain);

        const res = await fetch('/api/tunnel/start', {
            method: 'POST',
            body: body,
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Accept': 'application/json'
            }
        });
        const data = await this._parseResponseData(res);
        return { ok: res.ok && data.success, data };
    },

    /**
     * Stops the active DuckDNS tunnel and disables auto-start.
     */
    async stopTunnel() {
        const res = await fetch('/api/tunnel/stop', {
            method: 'POST',
            headers: { 'Accept': 'application/json' }
        });
        const data = res.ok ? {} : await this._parseResponseData(res);
        return { ok: res.ok, data };
    },

    /**
     * Validates DuckDNS credentials and DNS host resolution.
     */
    async testConnection(token, domain) {
        const body = new URLSearchParams();
        body.append('duckdns_token', token);
        body.append('duckdns_domain', domain);

        const res = await fetch('/api/tunnel/test', {
            method: 'POST',
            body: body,
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Accept': 'application/json'
            }
        });
        const data = await this._parseResponseData(res);
        return { ok: res.ok && data.success, data };
    },

    /**
     * Clears stored DuckDNS credentials and terminates the service.
     */
    async disconnect() {
        const res = await fetch('/api/tunnel/disconnect', {
            method: 'POST',
            headers: { 'Accept': 'application/json' }
        });
        const data = await this._parseResponseData(res);
        return { ok: res.ok, data };
    }
};

window.RemoteTunnelAPI = RemoteTunnelAPI;
