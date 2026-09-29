// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: web/static/js/remote_ui.js
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: UI feedback, clipboard copy, and DOM status renderer for remote tunnel management.

const RemoteUI = {
    showFeedback(message, type = 'success') {
        const feedbackBox = document.getElementById('remote-feedback');
        if (!feedbackBox) return;
        feedbackBox.className = `alert alert-${type} shadow-sm mb-4`;
        feedbackBox.innerHTML = message;
        feedbackBox.classList.remove('d-none');
        setTimeout(() => feedbackBox.classList.add('d-none'), 6000);
    },

    initClipboardCopy() {
        document.querySelectorAll('.copy-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const targetId = btn.getAttribute('data-target');
                const targetEl = document.getElementById(targetId);
                if (!targetEl) return;

                const textToCopy = (targetEl.tagName === 'INPUT' || targetEl.tagName === 'TEXTAREA')
                    ? targetEl.value
                    : (targetEl.textContent || targetEl.innerText);

                if (!textToCopy) return;

                try {
                    await navigator.clipboard.writeText(textToCopy);
                    const originalHTML = btn.innerHTML;
                    const copiedText = window._t ? window._t('common_copied', 'Copiado!') : 'Copiado!';
                    btn.innerHTML = `<i class="fa-solid fa-check text-success me-1"></i> ${copiedText}`;
                    btn.classList.add('btn-success');
                    btn.classList.remove('btn-outline-info', 'btn-outline-secondary', 'btn-outline-light');
                    setTimeout(() => {
                        btn.innerHTML = originalHTML;
                        btn.className = btn.className.replace('btn-success', '');
                        btn.classList.add(targetId === 'public-telemetry-url' ? 'btn-info' : 'btn-outline-secondary');
                    }, 2000);
                } catch (err) {
                    console.error('Failed to copy: ', err);
                    const copyFailedText = window._t ? window._t('common_copy_failed', 'Falha ao copiar para a área de transferência') : 'Falha ao copiar para a área de transferência';
                    RemoteUI.showFeedback(copyFailedText, 'danger');
                }
            });
        });
    },

    initTokenVisibilityToggle() {
        const btnToggleToken = document.getElementById('btn-toggle-token');
        const tokenInput = document.getElementById('duckdns_token');
        const iconToggleToken = document.getElementById('icon-toggle-token');
        if (btnToggleToken && tokenInput && iconToggleToken) {
            btnToggleToken.addEventListener('click', () => {
                const isPassword = tokenInput.type === 'password';
                tokenInput.type = isPassword ? 'text' : 'password';
                iconToggleToken.className = isPassword ? 'fa-solid fa-eye-slash text-warning' : 'fa-solid fa-eye';
            });
        }
    },

    updateTunnelUI(status, savedToken, savedDomain) {
        if (!status) return;

        const badgeTunnelState = document.getElementById('badge-tunnel-state');
        const badgeTunnelStateText = document.getElementById('badge-tunnel-state-text');
        const publicUrlInput = document.getElementById('public-telemetry-url');
        const errorBox = document.getElementById('tunnel-error-box');
        const errorMsg = document.getElementById('tunnel-error-msg');
        const btnConnectTunnel = document.getElementById('btn-connect-tunnel');
        const btnStopTunnel = document.getElementById('btn-stop-tunnel');
        const btnDisconnectTunnel = document.getElementById('btn-disconnect-tunnel');
        const startedTimeSpan = document.getElementById('tunnel-started-time');
        const ipv6Display = document.getElementById('ipv6-display');
        const onlineBanner = document.getElementById('tunnel-online-banner');
        const tokenInput = document.getElementById('duckdns_token');
        const domainInput = document.getElementById('duckdns_domain');

        // State Badge
        if (badgeTunnelState && badgeTunnelStateText) {
            badgeTunnelState.className = 'badge rounded-pill px-3 py-2 fs-6';
            badgeTunnelStateText.textContent = status.state;

            if (status.state === 'ONLINE') {
                badgeTunnelState.classList.add('bg-success');
                badgeTunnelState.innerHTML = '<i class="fa-solid fa-circle-check me-1"></i> ONLINE';
            } else if (status.state === 'CONNECTING') {
                badgeTunnelState.classList.add('bg-warning', 'text-dark');
                const connText = window._t ? window._t('remote_state_connecting', 'CONECTANDO') : 'CONECTANDO';
                badgeTunnelState.innerHTML = `<i class="fa-solid fa-spinner fa-spin me-1"></i> ${connText}`;
            } else if (status.state === 'ERROR') {
                badgeTunnelState.classList.add('bg-danger');
                const errText = window._t ? window._t('remote_state_error', 'ERRO') : 'ERRO';
                badgeTunnelState.innerHTML = `<i class="fa-solid fa-circle-exclamation me-1"></i> ${errText}`;
            } else {
                badgeTunnelState.classList.add('bg-secondary');
                const offText = window._t ? window._t('remote_state_offline', 'OFFLINE') : 'OFFLINE';
                badgeTunnelState.innerHTML = `<i class="fa-solid fa-circle-pause me-1"></i> ${offText}`;
            }
        }

        // IPv6 display
        if (ipv6Display) {
            if (status.ipv6_address) {
                ipv6Display.innerHTML = `<span class="badge bg-success-subtle text-success border border-success px-2 py-1"><i class="fa-solid fa-bolt me-1"></i> ${status.ipv6_address}</span>`;
            } else {
                const ipv6NotDetected = window._t ? window._t('remote_ipv6_not_detected', 'IPv6 não detectado. Operando via IPv4.') : 'IPv6 não detectado. Operando via IPv4.';
                ipv6Display.innerHTML = `<span class="text-muted"><i class="fa-solid fa-circle-info me-1"></i> ${ipv6NotDetected}</span>`;
            }
        }

        // Local port indicators
        const currentPort = status.local_port || '22100';
        const displayPort = status.use_https ? '443 (HTTPS)' : currentPort;
        const badgePort = document.getElementById('badge-local-port');
        if (badgePort) badgePort.textContent = displayPort;
        document.querySelectorAll('.local-port-label').forEach(el => el.textContent = displayPort);

        // Public URL
        if (status.state === 'ONLINE') {
            if (status.telemetry_url) {
                if (publicUrlInput) publicUrlInput.value = status.telemetry_url;
            } else if (status.domain) {
                const clean = status.domain.replace('.duckdns.org', '');
                const scheme = status.use_https ? 'https' : 'http';
                const portSuffix = status.use_https ? '' : `:${currentPort}`;
                if (publicUrlInput) publicUrlInput.value = `${scheme}://${clean}.duckdns.org${portSuffix}/api/telemetry`;
            }
        } else if (status.state === 'CONNECTING') {
            if (publicUrlInput) publicUrlInput.value = window._t ? window._t('remote_connecting', 'Conectando ao DuckDNS...') : 'Conectando ao DuckDNS...';
        } else {
            if (publicUrlInput) {
                const hasDomain = (domainInput && domainInput.value.trim()) || status.domain;
                publicUrlInput.value = hasDomain
                    ? (window._t ? window._t('remote_disconnected_hint', '[Desconectado] Inicie o DuckDNS para ativar a URL de telemetria') : '[Desconectado] Inicie o DuckDNS para ativar a URL de telemetria')
                    : (window._t ? window._t('remote_waiting_config', 'Aguardando configuração do DuckDNS...') : 'Aguardando configuração do DuckDNS...');
            }
        }

        // Error message box
        if (errorBox && errorMsg) {
            if (status.error_message) {
                errorMsg.textContent = status.error_message;
                errorBox.classList.remove('d-none');
            } else {
                errorBox.classList.add('d-none');
            }
        }

        // Action Buttons
        if (btnConnectTunnel && btnStopTunnel) {
            if (status.state === 'ONLINE') {
                btnConnectTunnel.classList.add('d-none');
                btnStopTunnel.classList.remove('d-none');
                btnStopTunnel.disabled = false;
                const discText = window._t ? window._t('remote_disconnect_btn', 'Desconectar DuckDNS') : 'Desconectar DuckDNS';
                btnStopTunnel.innerHTML = `<i class="fa-solid fa-power-off me-2"></i> ${discText}`;
            } else if (status.state === 'CONNECTING') {
                btnConnectTunnel.classList.remove('d-none');
                btnConnectTunnel.disabled = true;
                btnConnectTunnel.className = 'btn btn-warning text-dark py-2 fw-bold';
                const connText = window._t ? window._t('remote_connecting', 'Conectando...') : 'Conectando...';
                btnConnectTunnel.innerHTML = `<i class="fa-solid fa-spinner fa-spin me-2"></i> ${connText}`;
                btnStopTunnel.classList.add('d-none');
            } else {
                btnConnectTunnel.classList.remove('d-none');
                btnConnectTunnel.disabled = false;
                btnConnectTunnel.className = 'btn btn-success py-2 fw-bold';
                const connBtnText = window._t ? window._t('remote_connect_btn', 'Conectar DuckDNS') : 'Conectar DuckDNS';
                btnConnectTunnel.innerHTML = `<i class="fa-solid fa-play me-2"></i> ${connBtnText}`;
                btnStopTunnel.classList.add('d-none');
            }
        }

        // Disconnect button visibility
        if (btnDisconnectTunnel) {
            const hasCreds = (tokenInput && tokenInput.value.trim() !== '') || (savedToken !== '');
            btnDisconnectTunnel.classList.toggle('d-none', !hasCreds);
        }

        // Online banner
        if (onlineBanner) {
            onlineBanner.classList.toggle('d-none', status.state !== 'ONLINE');
        }

        if (startedTimeSpan && status.started_at) {
            startedTimeSpan.textContent = status.started_at;
        }
    }
};

window.RemoteUI = RemoteUI;
