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
// File: web/static/js/password_meter.js
// Author: Gabriel Moraes
// Date: 2026-10-06

(function (root, factory) {
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.PasswordStrengthMeter = factory();
    }
}(typeof self !== 'undefined' ? self : this, function () {

    const SPECIAL_CHARS_REGEX = /[!@#$%^&*()_\-+=\[\]{}|;:'",.<>?/`~\\§¬¢£]/;

    /**
     * Evaluates password complexity and returns criteria breakdown, score, and level.
     * @param {string} password 
     * @returns {object}
     */
    function evaluatePassword(password) {
        const pwd = password || '';
        const len = pwd.length;

        const criteria = {
            length: len >= 8,
            number: /[0-9]/.test(pwd),
            upper: /[A-Z]/.test(pwd),
            lower: /[a-z]/.test(pwd),
            special: SPECIAL_CHARS_REGEX.test(pwd) || /[^\w\s]/.test(pwd)
        };

        const isValid = criteria.length && criteria.number && criteria.upper && criteria.lower && criteria.special;

        if (len === 0) {
            return {
                valid: false,
                criteria,
                score: 0,
                percent: 0,
                level: 'empty',
                labelKey: 'password_strength_empty',
                defaultLabel: '—',
                barClass: 'bg-secondary',
                textClass: 'text-muted'
            };
        }

        // Calculate score
        let score = 0;

        // 1. Length scoring
        if (len < 8) {
            score += len * 4; // up to 28 pts
        } else if (len < 12) {
            score += 35 + (len - 8) * 5; // 35 - 50 pts
        } else {
            score += 55 + Math.min(25, (len - 12) * 3); // 55 - 80 pts
        }

        // 2. Character variety bonuses
        if (criteria.number) score += 10;
        if (criteria.upper) score += 10;
        if (criteria.lower) score += 10;
        if (criteria.special) score += 15;

        // 3. Extra bonus for long and complete passwords
        if (isValid && len >= 12) {
            score += 15;
        }

        // Cap score at 100
        score = Math.min(100, Math.max(5, score));

        // Determine level and visual states
        let level = 'very_weak';
        let labelKey = 'password_strength_very_weak';
        let defaultLabel = 'Muito Fraca';
        let barClass = 'bg-danger';
        let textClass = 'text-danger';
        let percent = Math.max(15, Math.min(30, score));

        if (!isValid) {
            if (len >= 8) {
                level = 'weak';
                labelKey = 'password_strength_weak';
                defaultLabel = 'Fraca (requisitos pendentes)';
                barClass = 'bg-warning';
                textClass = 'text-warning';
                percent = 40;
            } else {
                level = 'very_weak';
                labelKey = 'password_strength_very_weak';
                defaultLabel = 'Muito Fraca';
                barClass = 'bg-danger';
                textClass = 'text-danger';
                percent = Math.max(15, len * 4);
            }
        } else {
            // Valid password that satisfies all security criteria
            if (score >= 85) {
                level = 'very_strong';
                labelKey = 'password_strength_very_strong';
                defaultLabel = 'Excelente';
                barClass = 'bg-success';
                textClass = 'text-success';
                percent = 100;
            } else if (score >= 65) {
                level = 'strong';
                labelKey = 'password_strength_strong';
                defaultLabel = 'Forte';
                barClass = 'bg-info';
                textClass = 'text-info';
                percent = 80;
            } else {
                level = 'fair';
                labelKey = 'password_strength_fair';
                defaultLabel = 'Média / Segura';
                barClass = 'bg-warning';
                textClass = 'text-warning';
                percent = 60;
            }
        }

        return {
            valid: isValid,
            criteria,
            score,
            percent,
            level,
            labelKey,
            defaultLabel,
            barClass,
            textClass
        };
    }

    /**
     * Attaches interactive strength meter, checklist, and toggle visibility to DOM elements.
     * @param {object} options
     * @param {HTMLInputElement} options.passwordInput
     * @param {HTMLElement} [options.container]
     * @param {HTMLInputElement} [options.confirmInput]
     * @param {HTMLElement} [options.matchFeedback]
     */
    function attachMeter(options) {
        const input = options.passwordInput;
        if (!input) return null;

        const container = options.container || input.closest('form') || document;
        const bar = container.querySelector('.password-strength-bar');
        const label = container.querySelector('.password-strength-text');
        const reqItems = container.querySelectorAll('.req-item[data-req]');
        const toggleBtn = container.querySelector('.btn-toggle-password');
        const confirmInput = options.confirmInput;
        const matchFeedback = options.matchFeedback;

        function updateUI() {
            const val = input.value;
            const res = evaluatePassword(val);

            // Update Progress Bar
            if (bar) {
                bar.style.width = res.percent + '%';
                bar.className = 'progress-bar password-strength-bar transition-all ' + res.barClass;
                bar.setAttribute('aria-valuenow', res.percent);
            }

            // Update Label Text
            if (label) {
                const text = (window._t ? window._t(res.labelKey, res.defaultLabel) : res.defaultLabel);
                label.textContent = text;
                label.className = 'fw-bold password-strength-text ' + res.textClass;
            }

            // Update Checklist Items
            if (reqItems && reqItems.length > 0) {
                reqItems.forEach(item => {
                    const reqKey = item.getAttribute('data-req');
                    const isMet = !!res.criteria[reqKey];
                    const icon = item.querySelector('.req-icon');

                    if (isMet) {
                        item.classList.remove('text-muted');
                        item.classList.add('text-light');
                        if (icon) {
                            icon.className = 'fa-solid fa-circle-check text-success req-icon';
                        }
                    } else {
                        item.classList.remove('text-light');
                        item.classList.add('text-muted');
                        if (icon) {
                            icon.className = 'fa-solid fa-circle-xmark text-secondary req-icon';
                        }
                    }
                });
            }

            // Password confirmation check if confirm input exists
            if (confirmInput && matchFeedback) {
                const confirmVal = confirmInput.value;
                if (!confirmVal) {
                    matchFeedback.classList.add('d-none');
                } else if (confirmVal === val) {
                    matchFeedback.className = 'small mt-1 text-success d-flex align-items-center gap-1';
                    matchFeedback.innerHTML = '<i class="fa-solid fa-check"></i> <span>' + (window._t ? window._t('password_match_success', 'As senhas coincidem') : 'As senhas coincidem') + '</span>';
                    matchFeedback.classList.remove('d-none');
                } else {
                    matchFeedback.className = 'small mt-1 text-danger d-flex align-items-center gap-1';
                    matchFeedback.innerHTML = '<i class="fa-solid fa-xmark"></i> <span>' + (window._t ? window._t('auth_passwords_mismatch', 'As senhas não coincidem') : 'As senhas não coincidem') + '</span>';
                    matchFeedback.classList.remove('d-none');
                }
            }

            return res;
        }

        // Bind input events
        input.addEventListener('input', updateUI);
        input.addEventListener('change', updateUI);

        if (confirmInput) {
            confirmInput.addEventListener('input', updateUI);
            confirmInput.addEventListener('change', updateUI);
        }

        // Bind eye toggle button if found
        if (toggleBtn) {
            toggleBtn.addEventListener('click', (e) => {
                e.preventDefault();
                const currentType = input.getAttribute('type');
                const newType = currentType === 'password' ? 'text' : 'password';
                input.setAttribute('type', newType);
                const eyeIcon = toggleBtn.querySelector('i');
                if (eyeIcon) {
                    eyeIcon.className = newType === 'text' ? 'fa-solid fa-eye-slash' : 'fa-solid fa-eye';
                }
            });
        }

        // Initialize state
        updateUI();

        return {
            evaluate: () => evaluatePassword(input.value),
            isValid: () => evaluatePassword(input.value).valid,
            updateUI
        };
    }

    return {
        evaluate: evaluatePassword,
        attach: attachMeter
    };
}));
