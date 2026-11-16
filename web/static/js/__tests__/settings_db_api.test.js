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
// File: web/static/js/__tests__/settings_db_api.test.js
// Author: Gabriel Moraes
// Date: October 2026

import { describe, it, expect, vi, beforeEach } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

// Load vanilla JS file into test environment
const scriptCode = fs.readFileSync(path.resolve('web/static/js/settings_db_api.js'), 'utf-8');
const initDbClient = new Function(scriptCode + '; return new DatabaseApiClient();');

describe('DatabaseApiClient', () => {
    let client;

    beforeEach(() => {
        client = initDbClient();
        vi.restoreAllMocks();
    });

    it('getStatus returns parsed status from API', async () => {
        const mockStatus = {
            connected: true,
            type: 'postgres',
            latency_ms: 5,
        };

        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockStatus,
        });

        const status = await client.getStatus();
        expect(global.fetch).toHaveBeenCalledWith('/api/settings/database/status');
        expect(status.type).toBe('postgres');
        expect(status.connected).toBe(true);
    });

    it('getStatus throws error when response is not ok', async () => {
        global.fetch = vi.fn().mockResolvedValue({
            ok: false,
            status: 500,
        });

        await expect(client.getStatus()).rejects.toThrow('HTTP 500');
    });

    it('testConnection posts JSON config to test endpoint', async () => {
        const payload = {
            type: 'postgres',
            host: 'localhost',
            port: 5432,
            dbname: 'noxfort_db',
        };

        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => ({ connected: true, latency_ms: 4 }),
        });

        const res = await client.testConnection(payload);
        expect(res.ok).toBe(true);
        expect(global.fetch).toHaveBeenCalledWith(
            '/api/settings/database/test',
            expect.objectContaining({
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            })
        );
    });

    it('provisionUser sends creation request for database role', async () => {
        const payload = {
            host: 'localhost',
            admin_user: 'postgres',
            new_user: 'monitor_operator',
        };

        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => ({ success: true, message: 'User provisioned' }),
        });

        const res = await client.provisionUser(payload);
        expect(res.ok).toBe(true);
        expect(res.data.success).toBe(true);
    });

    it('saveConnection handles HTTP 403 JSON and plain text fallback cleanly', async () => {
        global.fetch = vi.fn().mockResolvedValueOnce({
            ok: false,
            status: 403,
            json: async () => ({ success: false, error: 'Acesso restrito' }),
        });

        const jsonRes = await client.saveConnection({ type: 'sqlite' });
        expect(jsonRes.ok).toBe(false);
        expect(jsonRes.data.error).toBe('Acesso restrito');

        global.fetch = vi.fn().mockResolvedValueOnce({
            ok: false,
            status: 403,
            json: async () => { throw new SyntaxError('Unexpected token'); },
            text: async () => 'Acesso restrito em texto puro',
        });

        const textRes = await client.saveConnection({ type: 'sqlite' });
        expect(textRes.ok).toBe(false);
        expect(textRes.data.error).toBe('Acesso restrito em texto puro');
    });

    it('gerencia listagem, criação e restauração de backups via API', async () => {
        global.fetch = vi.fn()
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({ backups: [{ filename: 'monitor_sqlite_new.db', size_bytes: 4096 }] }),
                text: async () => '',
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({ status: 'ok', filename: 'monitor_sqlite_new.db' }),
                text: async () => '',
            })
            .mockResolvedValueOnce({
                ok: true,
                status: 200,
                json: async () => ({ status: 'ok', message: 'Backup restaurado' }),
                text: async () => '',
            });

        const listRes = await client.listBackups();
        expect(listRes.backups).toHaveLength(1);
        expect(listRes.backups[0].filename).toBe('monitor_sqlite_new.db');
        expect(global.fetch).toHaveBeenNthCalledWith(1, '/api/settings/database/backups');

        const createRes = await client.createBackup();
        expect(createRes.ok).toBe(true);
        expect(global.fetch).toHaveBeenNthCalledWith(
            2,
            '/api/settings/database/backup',
            expect.objectContaining({ method: 'POST' }),
        );

        const restoreRes = await client.restoreBackup('monitor_sqlite_new.db');
        expect(restoreRes.ok).toBe(true);
        expect(global.fetch).toHaveBeenNthCalledWith(
            3,
            '/api/settings/database/restore',
            expect.objectContaining({
                method: 'POST',
                headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
                body: JSON.stringify({ filename: 'monitor_sqlite_new.db' }),
            }),
        );
    });
});
