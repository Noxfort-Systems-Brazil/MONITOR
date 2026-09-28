// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: web/static/js/remote_api.js
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: API client for DuckDNS remote access and tunnel orchestration endpoints.

const RemoteTunnelAPI = {
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
        const data = await res.json();
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
        const data = await res.json();
        return { ok: res.ok && data.success, data };
    },

    /**
     * Stops the active DuckDNS tunnel and disables auto-start.
     */
    async stopTunnel() {
        const res = await fetch('/api/tunnel/stop', { method: 'POST' });
        return { ok: res.ok };
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
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
        });
        const data = await res.json();
        return { ok: res.ok && data.success, data };
    },

    /**
     * Clears stored DuckDNS credentials and terminates the service.
     */
    async disconnect() {
        const res = await fetch('/api/tunnel/disconnect', { method: 'POST' });
        let data = {};
        try {
            data = await res.json();
        } catch (_) {}
        return { ok: res.ok, data };
    }
};

window.RemoteTunnelAPI = RemoteTunnelAPI;
