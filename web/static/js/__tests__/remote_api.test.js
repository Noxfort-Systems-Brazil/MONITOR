import { describe, it, expect, vi, beforeEach } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

// Load vanilla JS file into test environment
const scriptCode = fs.readFileSync(path.resolve('web/static/js/remote_api.js'), 'utf-8');
const initRemoteAPI = new Function(scriptCode + '; return RemoteTunnelAPI;');

describe('RemoteTunnelAPI Client', () => {
    let api;

    beforeEach(() => {
        api = initRemoteAPI();
        vi.restoreAllMocks();
    });

    it('getStatus retrieves tunnel status successfully', async () => {
        const mockResponse = {
            active: true,
            domain: 'noxfort.duckdns.org',
            ip: '192.168.1.100',
            status: 'online'
        };

        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockResponse,
        });

        const status = await api.getStatus();
        expect(global.fetch).toHaveBeenCalledWith('/api/tunnel/status');
        expect(status).toEqual(mockResponse);
    });

    it('getStatus returns null on network exception', async () => {
        global.fetch = vi.fn().mockRejectedValue(new Error('Network error'));
        const status = await api.getStatus();
        expect(status).toBeNull();
    });

    it('saveConfig submits url-encoded credentials', async () => {
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => ({ success: true, message: 'Saved' }),
        });

        const res = await api.saveConfig('my-token', 'my-subdomain');
        expect(res.ok).toBe(true);
        expect(global.fetch).toHaveBeenCalledWith(
            '/api/tunnel/save',
            expect.objectContaining({
                method: 'POST',
                headers: expect.objectContaining({
                    'Content-Type': 'application/x-www-form-urlencoded',
                }),
            })
        );
    });

    it('startTunnel starts duckdns service', async () => {
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => ({ success: true, url: 'https://test.duckdns.org' }),
        });

        const res = await api.startTunnel('my-token', 'test');
        expect(res.ok).toBe(true);
        expect(res.data.url).toBe('https://test.duckdns.org');
    });
});
