// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: web/static/js/remote.js
// Author: Gabriel Moraes
// Date: 2026-09-13
// Modified: 2026-09-14 (Modularized into remote_api.js and remote_ui.js)

document.addEventListener('DOMContentLoaded', () => {
    const api = window.RemoteTunnelAPI;
    const ui = window.RemoteUI;

    // Initialize helpers
    ui.initClipboardCopy();
    ui.initTokenVisibilityToggle();

    // Check URL parameters for ?success=1
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('success') === '1') {
        ui.showFeedback('✅ Configurações salvas com sucesso! O serviço DuckDNS está ativo.', 'success');
        window.history.replaceState({}, document.title, window.location.pathname);
    }

    // Input elements & saved credentials cache
    const tokenInput = document.getElementById('duckdns_token');
    const domainInput = document.getElementById('duckdns_domain');
    let savedTokenValue = tokenInput ? tokenInput.value.trim() : '';
    let savedDomainValue = domainInput ? domainInput.value.trim() : '';

    // Action buttons & forms
    const btnConnectTunnel = document.getElementById('btn-connect-tunnel');
    const btnSaveCredentials = document.getElementById('btn-save-credentials');
    const btnStopTunnel = document.getElementById('btn-stop-tunnel');
    const btnTestTunnel = document.getElementById('btn-test-tunnel');
    const iconTestTunnel = document.getElementById('icon-test-tunnel');
    const textTestTunnel = document.getElementById('text-test-tunnel');
    const btnDisconnectTunnel = document.getElementById('btn-disconnect-tunnel');
    const btnRefresh = document.getElementById('btn-refresh-status');
    const tunnelForm = document.getElementById('tunnelConfigForm');

    async function checkStatus() {
        const data = await api.getStatus();
        if (data) {
            ui.updateTunnelUI(data, savedTokenValue, savedDomainValue);
            return data;
        }
        return null;
    }

    // Refresh Button
    if (btnRefresh) {
        btnRefresh.addEventListener('click', async () => {
            btnRefresh.disabled = true;
            btnRefresh.innerHTML = '<i class="fa-solid fa-arrows-rotate fa-spin me-1"></i> Verificando...';
            await checkStatus();
            setTimeout(() => {
                btnRefresh.disabled = false;
                btnRefresh.innerHTML = '<i class="fa-solid fa-arrows-rotate me-1"></i> Atualizar';
            }, 800);
        });
    }

    // Form Submit: Salvar Credenciais
    if (tunnelForm) {
        tunnelForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const tokenVal = tokenInput ? tokenInput.value.trim() : '';
            const domVal = domainInput ? domainInput.value.trim() : '';
            if (!tokenVal || !domVal) {
                ui.showFeedback('⚠️ Por favor, preencha o <strong>Token</strong> e o <strong>Subdomínio</strong> DuckDNS.', 'warning');
                return;
            }

            if (btnSaveCredentials) {
                btnSaveCredentials.disabled = true;
                btnSaveCredentials.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Salvando...';
            }

            try {
                const { ok, data } = await api.saveConfig(tokenVal, domVal);
                if (ok) {
                    savedTokenValue = tokenVal;
                    savedDomainValue = domVal;
                    ui.showFeedback('✅ <strong>Credenciais salvas com sucesso!</strong> O serviço permanece desconectado até você clicar em "Conectar DuckDNS".', 'success');
                    if (btnDisconnectTunnel) btnDisconnectTunnel.classList.remove('d-none');
                } else {
                    ui.showFeedback(`Falha ao salvar: ${data?.error || 'Erro desconhecido'}`, 'danger');
                }
            } catch (err) {
                ui.showFeedback('Erro de comunicação: ' + err.message, 'danger');
            } finally {
                if (btnSaveCredentials) {
                    btnSaveCredentials.disabled = false;
                    btnSaveCredentials.innerHTML = '<i class="fa-solid fa-floppy-disk me-2"></i> Salvar Credenciais';
                }
            }
        });
    }

    // Connect Button
    if (btnConnectTunnel) {
        btnConnectTunnel.addEventListener('click', async () => {
            const tokenVal = tokenInput ? tokenInput.value.trim() : '';
            const domVal = domainInput ? domainInput.value.trim() : '';
            if (!tokenVal || !domVal) {
                ui.showFeedback('⚠️ Por favor, preencha o <strong>Token</strong> e o <strong>Subdomínio</strong> DuckDNS para conectar.', 'warning');
                return;
            }

            btnConnectTunnel.disabled = true;
            btnConnectTunnel.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Conectando...';

            try {
                const { ok, data } = await api.startTunnel(tokenVal, domVal);
                if (ok) {
                    ui.showFeedback('Iniciando serviço DuckDNS...', 'info');
                    savedTokenValue = tokenVal;
                    savedDomainValue = domVal;
                    ui.updateTunnelUI({ state: 'CONNECTING', domain: savedDomainValue, telemetry_url: '' }, savedTokenValue, savedDomainValue);
                    pollStatusUntilOnline();
                } else {
                    ui.showFeedback(`Falha ao conectar: ${data?.error || 'Erro desconhecido'}`, 'danger');
                    btnConnectTunnel.disabled = false;
                    btnConnectTunnel.innerHTML = '<i class="fa-solid fa-play me-2"></i> Conectar DuckDNS';
                }
            } catch (err) {
                ui.showFeedback('Erro de comunicação: ' + err.message, 'danger');
                btnConnectTunnel.disabled = false;
                btnConnectTunnel.innerHTML = '<i class="fa-solid fa-play me-2"></i> Conectar DuckDNS';
            }
        });
    }

    // Disconnect / Stop Button
    if (btnStopTunnel) {
        btnStopTunnel.addEventListener('click', async () => {
            btnStopTunnel.disabled = true;
            btnStopTunnel.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-2"></i> Desconectando...';
            try {
                const { ok } = await api.stopTunnel();
                if (ok) {
                    ui.showFeedback('Serviço DuckDNS desconectado com sucesso. O serviço permanecerá desconectado ao fechar o programa.', 'secondary');
                    await checkStatus();
                } else {
                    ui.showFeedback('Erro ao desconectar DuckDNS.', 'danger');
                    btnStopTunnel.disabled = false;
                    btnStopTunnel.innerHTML = '<i class="fa-solid fa-power-off me-2"></i> Desconectar DuckDNS';
                }
            } catch (err) {
                ui.showFeedback('Erro de rede: ' + err.message, 'danger');
                btnStopTunnel.disabled = false;
                btnStopTunnel.innerHTML = '<i class="fa-solid fa-power-off me-2"></i> Desconectar DuckDNS';
            }
        });
    }

    // Test DuckDNS Connection Button
    if (btnTestTunnel) {
        btnTestTunnel.addEventListener('click', async () => {
            const tokenVal = tokenInput ? tokenInput.value.trim() : '';
            const domVal = domainInput ? domainInput.value.trim() : '';
            if (!tokenVal || !domVal) {
                ui.showFeedback('⚠️ Preencha o <strong>Token</strong> e o <strong>Subdomínio</strong> DuckDNS para realizar o teste de conexão.', 'warning');
                return;
            }

            btnTestTunnel.disabled = true;
            if (iconTestTunnel) iconTestTunnel.className = 'fa-solid fa-spinner fa-spin me-2';
            if (textTestTunnel) textTestTunnel.textContent = 'Testando Conectividade...';

            try {
                const { ok, data } = await api.testConnection(tokenVal, domVal);
                if (ok) {
                    const ipInfo = (data.resolved_ips && data.resolved_ips.length > 0)
                        ? ` (IPs resolvidos: <code>${data.resolved_ips.join(', ')}</code>)`
                        : '';
                    ui.showFeedback(`✅ <strong>DuckDNS Funcionando!</strong> ${data.message}${ipInfo}`, 'success');
                } else {
                    ui.showFeedback(`❌ <strong>Falha no Teste:</strong> ${data?.error || 'Erro desconhecido ao testar DuckDNS.'}`, 'danger');
                }
            } catch (err) {
                ui.showFeedback('❌ <strong>Erro de comunicação:</strong> ' + err.message, 'danger');
            } finally {
                btnTestTunnel.disabled = false;
                if (iconTestTunnel) iconTestTunnel.className = 'fa-solid fa-stethoscope me-2';
                if (textTestTunnel) textTestTunnel.textContent = 'Testar Conexão DuckDNS';
            }
        });
    }

    // Clear Credentials Button
    if (btnDisconnectTunnel) {
        btnDisconnectTunnel.addEventListener('click', async () => {
            if (!confirm('Deseja realmente remover as credenciais do DuckDNS e desativar o serviço?')) {
                return;
            }
            btnDisconnectTunnel.disabled = true;
            try {
                const { ok, data } = await api.disconnect();
                if (ok) {
                    if (tokenInput) tokenInput.value = '';
                    if (domainInput) domainInput.value = '';
                    savedTokenValue = '';
                    savedDomainValue = '';
                    ui.showFeedback('Credenciais removidas e serviço DuckDNS desativado com sucesso.', 'info');
                    await checkStatus();
                } else {
                    ui.showFeedback('Erro ao remover credenciais: ' + (data?.error || 'Erro desconhecido'), 'danger');
                }
            } catch (err) {
                ui.showFeedback('Erro de rede: ' + err.message, 'danger');
            } finally {
                btnDisconnectTunnel.disabled = false;
            }
        });
    }

    async function pollStatusUntilOnline() {
        let attempts = 0;
        const interval = setInterval(async () => {
            attempts++;
            const status = await checkStatus();
            if (status) {
                if (status.state === 'ONLINE') {
                    clearInterval(interval);
                    ui.showFeedback('✅ Domínio DuckDNS registrado com sucesso!', 'success');
                } else if (status.state === 'ERROR' || attempts >= 15) {
                    clearInterval(interval);
                    if (status.state === 'ERROR') {
                        ui.showFeedback(`Erro ao registrar: ${status.error_message || 'Verifique Token e Subdomínio'}`, 'danger');
                    }
                }
            }
        }, 1200);
    }

    // Initial status check & periodic poller
    checkStatus();
    const livePoller = setInterval(checkStatus, 4000);
    window.addEventListener('beforeunload', () => clearInterval(livePoller));
});
