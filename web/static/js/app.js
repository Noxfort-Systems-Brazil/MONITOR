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
// File: web/static/js/app.js
// Author: Gabriel Moraes
// Date: October 2026

document.addEventListener("DOMContentLoaded", function () {
    // 1. Initialize Bootstrap Tooltips
    // Select all elements with data-bs-toggle="tooltip"
    var tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    var tooltipList = tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });

    // 2. Active Link Highlighting
    // Automatically marks the nav link corresponding to the current URL as active
    const currentPath = window.location.pathname;
    const navLinks = document.querySelectorAll('.navbar-nav .nav-link');

    navLinks.forEach(link => {
        if (link.getAttribute('href') === currentPath) {
            link.classList.add('active');
            link.classList.add('fw-bold'); // Make it bold for better visibility
        }
    });

    console.log("Noxfort Monitor UI Loaded.");

    // 2b. Authenticated Self-Service Password Update
    const btnSelfPass = document.getElementById('btn-submit-self-pass');
    if (btnSelfPass) {
        btnSelfPass.addEventListener('click', async () => {
            const alertEl = document.getElementById('self-pass-alert');
            const username = document.getElementById('self_pass_username')?.value || '';
            const currentPassword = document.getElementById('self_current_password')?.value || '';
            const newPassword = document.getElementById('self_new_password')?.value || '';
            if (!currentPassword || !newPassword) return;

            btnSelfPass.disabled = true;
            try {
                const fd = new FormData();
                fd.append('username', username);
                fd.append('current_password', currentPassword);
                fd.append('new_password', newPassword);
                const res = await fetch('/api/users/password', { method: 'POST', body: fd });
                const data = await res.json();
                if (alertEl) {
                    alertEl.className = res.ok && data.success ? 'alert alert-success small mb-3' : 'alert alert-danger small mb-3';
                    alertEl.textContent = (res.ok && data.success ? '✅ ' : '❌ ') + (data.message || 'Erro ao atualizar senha.');
                    alertEl.classList.remove('d-none');
                }
                if (res.ok && data.success) {
                    setTimeout(() => window.location.replace('/login'), 1000);
                }
            } catch (err) {
                if (alertEl) {
                    alertEl.className = 'alert alert-danger small mb-3';
                    alertEl.textContent = '❌ Falha de comunicação: ' + err.message;
                    alertEl.classList.remove('d-none');
                }
            } finally {
                btnSelfPass.disabled = false;
            }
        });
    }
});

// 3. Native & Web Fullscreen Handler (F11 to toggle, Escape to exit)
window.addEventListener("keydown", function (e) {
    if (e.key === "F11") {
        e.preventDefault();
        fetch("/api/window/toggle-fullscreen", { method: "POST" })
            .then(res => res.json())
            .then(data => {
                // If running in a standard web browser (non-desktop container), fallback to HTML5 Fullscreen API
                if (data && data.desktop === false) {
                    if (!document.fullscreenElement) {
                        if (document.documentElement.requestFullscreen) {
                            document.documentElement.requestFullscreen().catch(() => {});
                        }
                    } else {
                        if (document.exitFullscreen) {
                            document.exitFullscreen().catch(() => {});
                        }
                    }
                }
            })
            .catch(() => {
                // Network error or offline fallback
                if (!document.fullscreenElement) {
                    if (document.documentElement.requestFullscreen) {
                        document.documentElement.requestFullscreen().catch(() => {});
                    }
                } else {
                    if (document.exitFullscreen) {
                        document.exitFullscreen().catch(() => {});
                    }
                }
            });
    } else if (e.key === "Escape") {
        fetch("/api/window/exit-fullscreen", { method: "POST" }).catch(() => {});
        if (document.fullscreenElement && document.exitFullscreen) {
            document.exitFullscreen().catch(() => {});
        }
    }
});
