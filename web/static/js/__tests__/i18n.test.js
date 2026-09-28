import { describe, it, expect, vi, beforeEach } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

const scriptCode = fs.readFileSync(path.resolve('web/static/js/i18n.js'), 'utf-8');
const createI18nManager = new Function(scriptCode + '; return new I18nManager();');

describe('I18nManager Engine', () => {
    let i18n;

    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
        document.documentElement.lang = '';
        vi.restoreAllMocks();
    });

    it('detects language from localStorage if valid', () => {
        localStorage.setItem('noxfort_language', 'es');
        i18n = createI18nManager();
        expect(i18n.currentLang).toBe('es');
    });

    it('detects browser language if localStorage is unset or invalid', () => {
        localStorage.setItem('noxfort_language', 'invalid_code');
        i18n = createI18nManager();
        expect(i18n.currentLang).toBe('en');
    });

    it('falls back to default language (pt) if both localStorage and browser are unsupported', () => {
        Object.defineProperty(navigator, 'language', { value: 'de-DE', configurable: true });
        localStorage.setItem('noxfort_language', 'invalid_code');
        i18n = createI18nManager();
        expect(i18n.currentLang).toBe('pt');
    });

    it('applies translations to DOM data-i18n elements', async () => {
        document.body.innerHTML = `
            <h1 data-i18n="app_title">Default Title</h1>
            <input data-i18n-placeholder="search_placeholder" placeholder="Default Search" />
        `;

        i18n = createI18nManager();
        i18n.translations = {
            app_title: 'Noxfort Monitor Industrial',
            search_placeholder: 'Pesquisar dispositivos...',
        };

        i18n.applyTranslations();

        const title = document.querySelector('[data-i18n="app_title"]');
        const input = document.querySelector('[data-i18n-placeholder]');

        expect(title.textContent).toBe('Noxfort Monitor Industrial');
        expect(input.getAttribute('placeholder')).toBe('Pesquisar dispositivos...');
        expect(window._t('app_title')).toBe('Noxfort Monitor Industrial');
        expect(window._t('unknown_key')).toBe('unknown_key');
    });

    it('loads translation dictionary via fetch', async () => {
        const mockDict = { dashboard: 'Painel Geral' };
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockDict,
        });

        i18n = createI18nManager();
        i18n.currentLang = 'pt';
        await i18n.loadTranslations();

        expect(global.fetch).toHaveBeenCalledWith('/static/locales/pt.json');
        expect(i18n.translations.dashboard).toBe('Painel Geral');
    });
});
