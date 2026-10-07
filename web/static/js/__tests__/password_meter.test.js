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
// File: web/static/js/__tests__/password_meter.test.js
// Author: Gabriel Moraes
// Date: 2026-10-06

import { describe, it, expect, beforeEach } from 'vitest';
import PasswordStrengthMeter from '../password_meter.js';

describe('PasswordStrengthMeter Engine', () => {
    describe('evaluate()', () => {
        it('returns empty level and 0 score for empty string', () => {
            const res = PasswordStrengthMeter.evaluate('');
            expect(res.valid).toBe(false);
            expect(res.score).toBe(0);
            expect(res.level).toBe('empty');
            expect(res.criteria.length).toBe(false);
            expect(res.criteria.number).toBe(false);
            expect(res.criteria.upper).toBe(false);
            expect(res.criteria.lower).toBe(false);
            expect(res.criteria.special).toBe(false);
        });

        it('identifies missing length (< 8 chars)', () => {
            const res = PasswordStrengthMeter.evaluate('Ab1!xyz');
            expect(res.valid).toBe(false);
            expect(res.criteria.length).toBe(false);
            expect(res.criteria.number).toBe(true);
            expect(res.criteria.upper).toBe(true);
            expect(res.criteria.lower).toBe(true);
            expect(res.criteria.special).toBe(true);
            expect(res.level).toBe('very_weak');
        });

        it('identifies missing number', () => {
            const res = PasswordStrengthMeter.evaluate('Abcdefgh!');
            expect(res.valid).toBe(false);
            expect(res.criteria.number).toBe(false);
            expect(res.criteria.length).toBe(true);
            expect(res.criteria.upper).toBe(true);
            expect(res.criteria.lower).toBe(true);
            expect(res.criteria.special).toBe(true);
            expect(res.level).toBe('weak');
        });

        it('identifies missing uppercase letter', () => {
            const res = PasswordStrengthMeter.evaluate('abcdef123!@');
            expect(res.valid).toBe(false);
            expect(res.criteria.upper).toBe(false);
            expect(res.level).toBe('weak');
        });

        it('identifies missing lowercase letter', () => {
            const res = PasswordStrengthMeter.evaluate('ABCDEF123!@');
            expect(res.valid).toBe(false);
            expect(res.criteria.lower).toBe(false);
            expect(res.level).toBe('weak');
        });

        it('identifies missing special character', () => {
            const res = PasswordStrengthMeter.evaluate('Abcdef1234');
            expect(res.valid).toBe(false);
            expect(res.criteria.special).toBe(false);
            expect(res.level).toBe('weak');
        });

        it('validates compliant password (8+ chars, upper, lower, number, special)', () => {
            const res = PasswordStrengthMeter.evaluate('Admin@2026');
            expect(res.valid).toBe(true);
            expect(res.criteria.length).toBe(true);
            expect(res.criteria.number).toBe(true);
            expect(res.criteria.upper).toBe(true);
            expect(res.criteria.lower).toBe(true);
            expect(res.criteria.special).toBe(true);
            expect(['fair', 'strong', 'very_strong']).toContain(res.level);
            expect(res.score).toBeGreaterThanOrEqual(60);
        });

        it('rewards long and highly complex passwords with excellent/very_strong rating', () => {
            const res = PasswordStrengthMeter.evaluate('SuperSecur3#Password!2026');
            expect(res.valid).toBe(true);
            expect(res.level).toBe('very_strong');
            expect(res.percent).toBe(100);
            expect(res.barClass).toBe('bg-success');
        });
    });

    describe('attach() DOM integration', () => {
        beforeEach(() => {
            document.body.innerHTML = `
                <form id="testForm">
                    <div class="input-group">
                        <input type="password" id="pass" value="" />
                        <button class="btn-toggle-password"><i class="fa-solid fa-eye"></i></button>
                    </div>
                    <div class="password-strength-bar"></div>
                    <div class="password-strength-text"></div>
                    <div class="req-item" data-req="length"><i class="req-icon"></i></div>
                    <div class="req-item" data-req="number"><i class="req-icon"></i></div>
                    <div class="req-item" data-req="upper"><i class="req-icon"></i></div>
                    <div class="req-item" data-req="lower"><i class="req-icon"></i></div>
                    <div class="req-item" data-req="special"><i class="req-icon"></i></div>
                    <input type="password" id="pass_confirm" value="" />
                    <div id="matchFeedback"></div>
                </form>
            `;
        });

        it('binds to DOM elements and dynamically updates UI on input event', () => {
            const passInput = document.getElementById('pass');
            const form = document.getElementById('testForm');
            const handler = PasswordStrengthMeter.attach({
                passwordInput: passInput,
                container: form
            });

            expect(handler).not.toBeNull();
            expect(handler.isValid()).toBe(false);

            passInput.value = 'Noxfort@2026';
            passInput.dispatchEvent(new Event('input'));

            expect(handler.isValid()).toBe(true);

            const lengthItem = form.querySelector('[data-req="length"]');
            expect(lengthItem.classList.contains('text-light')).toBe(true);
            const icon = lengthItem.querySelector('.req-icon');
            expect(icon.classList.contains('fa-circle-check')).toBe(true);
        });

        it('toggles password visibility when toggle button is clicked', () => {
            const passInput = document.getElementById('pass');
            const form = document.getElementById('testForm');
            const toggleBtn = form.querySelector('.btn-toggle-password');

            PasswordStrengthMeter.attach({
                passwordInput: passInput,
                container: form
            });

            expect(passInput.getAttribute('type')).toBe('password');

            toggleBtn.click();
            expect(passInput.getAttribute('type')).toBe('text');
            expect(toggleBtn.querySelector('i').classList.contains('fa-eye-slash')).toBe(true);

            toggleBtn.click();
            expect(passInput.getAttribute('type')).toBe('password');
            expect(toggleBtn.querySelector('i').classList.contains('fa-eye')).toBe(true);
        });
    });
});
