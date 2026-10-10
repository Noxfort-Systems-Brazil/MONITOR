/**
 * Noxfort Monitor™ - Automated Presentation Video Generator
 * Copyright (C) 2026 Noxfort Systems
 *
 * Gera gravação de alta fidelidade (1080p @ 60 FPS) do Noxfort Monitor
 * navegando ABA POR ABA, simulando cadastro de usuário ao vivo com
 * medidor de força de senha, injeção de incidente crítico na rede e
 * visualização da trilha de auditoria para apresentações executivas.
 */

const { chromium } = require('playwright');
const { spawn, execSync } = require('child_process');
const http = require('http');
const fs = require('fs');
const path = require('path');
const {
  seedNormalTelemetry,
  triggerCriticalIncident,
  triggerRecovery
} = require('./mock_telemetry.cjs');

const ROOT_DIR = path.resolve(__dirname, '../..');
const OUTPUT_DIR = path.join(ROOT_DIR, 'recordings');
const FINAL_MP4 = path.join(OUTPUT_DIR, 'noxfort_monitor_apresentacao.mp4');
const SERVER_URL = 'http://127.0.0.1:22100';

// Configuração de tempos de exibição (em milissegundos)
const TIMINGS = {
  loginPre: 1200,          // Pausa antes de digitar credenciais
  typeDelay: 45,           // Velocidade de digitação natural
  loginPost: 1500,         // Pausa após autenticação
  tabOverview: 5500,       // 1. Dashboard inicial
  tabDevices: 5000,        // 2. Dispositivos
  tabContacts: 5000,       // 3. Equipe de Resposta
  tabRemote: 5000,         // 4. Acesso Remoto DuckDNS
  tabServer: 5000,         // 5. Servidor Dual-Engine
  tabSettings: 5000,       // 6. Configurações MQTT/SMTP
  tabUsersPre: 1500,       // 7. Pausa ao abrir Usuários
  tabUsersPost: 4000,      // Pausa após criar o usuário
  incidentDashboard: 7000, // 8. Dashboard reagindo ao incidente ao vivo
  auditSecurity: 4000,     // 9. Logs de segurança da auditoria
  auditAlerts: 4000,       // Logs de despacho de alertas (SLA)
  auditTransitions: 3500,  // Logs de transições de uptime
  closing: 3000            // Encerramento nominal
};

function loadCredentials() {
  const envPath = path.join(ROOT_DIR, '.env');
  let user = 'admin';
  let pass = 'Admin@Noxfort2026!';
  if (fs.existsSync(envPath)) {
    const lines = fs.readFileSync(envPath, 'utf8').split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed.startsWith('MONITOR_ADMIN_USER=')) {
        user = trimmed.split('=')[1].trim();
      } else if (trimmed.startsWith('MONITOR_ADMIN_PASSWORD=')) {
        pass = trimmed.split('=')[1].trim();
      }
    }
  }
  return { user, pass };
}

function cleanupDemoUser() {
  const dbPath = path.join(process.env.HOME || '', 'Documentos/Monitor/monitor_logs.db');
  if (fs.existsSync(dbPath)) {
    try {
      execSync(`sqlite3 "${dbPath}" "DELETE FROM users WHERE username = 'op_sala_central';" 2>/dev/null || true`);
      console.log('🧹 Limpeza prévia: usuário de demonstração "op_sala_central" redefinido para novo cadastro.');
    } catch (_) {}
  }
}

function checkServerReady() {
  return new Promise((resolve) => {
    const req = http.get(`${SERVER_URL}/login`, (res) => {
      resolve(res.statusCode === 200 || res.statusCode === 303);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(1000, () => {
      req.destroy();
      resolve(false);
    });
  });
}

async function waitForServer(maxAttempts = 30) {
  for (let i = 0; i < maxAttempts; i++) {
    const ready = await checkServerReady();
    if (ready) return true;
    await new Promise((r) => setTimeout(r, 500));
  }
  return false;
}

async function main() {
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  }

  cleanupDemoUser();

  const creds = loadCredentials();
  console.log(`🔑 Credenciais carregadas: Usuário '${creds.user}'`);

  let serverProcess = null;
  const isRunning = await checkServerReady();

  if (!isRunning) {
    console.log('⚡ Iniciando Noxfort Monitor em modo DEMO (:22100)...');
    const binPath = path.join(ROOT_DIR, 'bin/noxfort-monitor');
    if (!fs.existsSync(binPath)) {
      execSync('make build', { cwd: ROOT_DIR, stdio: 'inherit' });
    }

    serverProcess = spawn(binPath, ['--demo'], {
      cwd: ROOT_DIR,
      stdio: 'inherit'
    });

    const ready = await waitForServer();
    if (!ready) {
      if (serverProcess) serverProcess.kill();
      throw new Error('Falha ao inicializar servidor do Noxfort Monitor em 15 segundos.');
    }
  }

  console.log('📡 Injetando telemetrias de base (Synapse, Carina, Hardware)...');
  await seedNormalTelemetry();

  console.log('🚀 Inicializando Chromium Playwright (1920x1080 Full HD)...');
  const browser = await chromium.launch({
    headless: true,
    args: ['--hide-scrollbars', '--disable-gpu', '--no-sandbox']
  });

  const context = await browser.newContext({
    recordVideo: {
      dir: OUTPUT_DIR,
      size: { width: 1920, height: 1080 }
    },
    viewport: { width: 1920, height: 1080 }
  });

  const page = await context.newPage();

  try {
    // ══════════════════════════════════════════════════════════════════════
    // CENA 0: LOGIN & AUTENTICAÇÃO SEGURA
    // ══════════════════════════════════════════════════════════════════════
    console.log('🔐 [Cena 0] Acessando tela de autenticação segura (/login)...');
    await page.goto(`${SERVER_URL}/login`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(TIMINGS.loginPre);

    console.log('✍️  Digitando credenciais do Administrador...');
    await page.type('#username', creds.user, { delay: TIMINGS.typeDelay });
    await page.type('#password', creds.pass, { delay: TIMINGS.typeDelay });
    await page.click('#btn-submit');

    await page.waitForURL((url) => url.pathname === '/', { timeout: 15000 });
    await page.waitForTimeout(TIMINGS.loginPost);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 1: OVERVIEW / DASHBOARD OPERACIONAL
    // ══════════════════════════════════════════════════════════════════════
    console.log('📊 [Aba 1/8] Overview / Dashboard (Visão Operacional)...');
    await page.waitForTimeout(TIMINGS.tabOverview);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 2: DEVICES (DISPOSITIVOS & AGENTES IOT)
    // ══════════════════════════════════════════════════════════════════════
    console.log('🖥️  [Aba 2/8] Dispositivos & Agentes de Borda (/devices)...');
    await page.click('a[href="/devices"]');
    await page.waitForURL((url) => url.pathname === '/devices');
    await page.waitForTimeout(TIMINGS.tabDevices);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 3: RESPONSE TEAM / CONTATOS
    // ══════════════════════════════════════════════════════════════════════
    console.log('👥 [Aba 3/8] Matriz de Resposta a Incidentes (/contacts)...');
    await page.click('a[href="/contacts"]');
    await page.waitForURL((url) => url.pathname === '/contacts');
    await page.waitForTimeout(TIMINGS.tabContacts);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 4: REMOTE ACCESS (DUCKDNS & CONECTIVIDADE WAN)
    // ══════════════════════════════════════════════════════════════════════
    console.log('🌐 [Aba 4/8] Acesso Remoto & Conectividade WAN (/remote)...');
    await page.click('a[href="/remote"]');
    await page.waitForURL((url) => url.pathname === '/remote');
    await page.waitForTimeout(TIMINGS.tabRemote);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 5: SERVER & DUAL-ENGINE PERSISTENCE
    // ══════════════════════════════════════════════════════════════════════
    console.log('🗄️  [Aba 5/8] Servidor & Arquitetura Dual-Engine (/server)...');
    await page.click('a[href="/server"]');
    await page.waitForURL((url) => url.pathname === '/server');
    await page.waitForTimeout(TIMINGS.tabServer);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 6: SETTINGS (MQTT, SMTP, TELEGRAM)
    // ══════════════════════════════════════════════════════════════════════
    console.log('⚙️  [Aba 6/8] Configurações de Notificação & Broker (/settings)...');
    await page.click('a[href="/settings"]');
    await page.waitForURL((url) => url.pathname === '/settings');
    await page.waitForTimeout(TIMINGS.tabSettings);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 7: USERS & SIMULAÇÃO DE CADASTRO DE OPERADOR AO VIVO
    // ══════════════════════════════════════════════════════════════════════
    console.log('👤 [Aba 7/8] Gerenciamento de Usuários & Cadastro ao Vivo (/users)...');
    await page.click('a[href="/users"]');
    await page.waitForURL((url) => url.pathname === '/users');
    await page.waitForTimeout(TIMINGS.tabUsersPre);

    console.log('✍️  Digitando nome do novo operador ("op_sala_central")...');
    await page.type('#new_username', 'op_sala_central', { delay: TIMINGS.typeDelay });

    console.log('🔒 Digitando senha forte com animação do medidor de segurança...');
    await page.type('#new_password', 'Operador@2026!Sec', { delay: TIMINGS.typeDelay });
    await page.waitForTimeout(800);

    console.log('➕ Submetendo formulário de criação de conta...');
    await page.click('#btn-add-user');
    await page.waitForTimeout(TIMINGS.tabUsersPost);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 8: INCIDENTE AO VIVO & REAÇÃO DO DASHBOARD
    // ══════════════════════════════════════════════════════════════════════
    console.log('🚨 [Cena Incidente] DISPARANDO INCIDENTE CRÍTICO AO VIVO NA REDE...');
    await triggerCriticalIncident();

    console.log('⚡ Retornando ao Dashboard para capturar a reação em tempo real...');
    await page.click('a[href="/"]');
    await page.waitForURL((url) => url.pathname === '/');
    await page.waitForTimeout(TIMINGS.incidentDashboard);

    // ══════════════════════════════════════════════════════════════════════
    // ABA 9: TRILHA DE AUDITORIA TRIPLA (SECURITY, ALERTS, AVAILABILITY)
    // ══════════════════════════════════════════════════════════════════════
    console.log('🛡️  [Aba 8/8] Trilha de Auditoria Tripla (/audit)...');
    await page.click('a[href="/audit"]');
    await page.waitForURL((url) => url.pathname === '/audit');
    await page.waitForTimeout(TIMINGS.auditSecurity);

    console.log('🔔 Alternando para aba de Alertas Despachados (SLA)...');
    const alertsTab = await page.$('#alerts-tab');
    if (alertsTab) {
      await alertsTab.click();
      await page.waitForTimeout(TIMINGS.auditAlerts);
    }

    const transitionsTab = await page.$('#transitions-tab');
    if (transitionsTab) {
      console.log('📈 Alternando para aba de Transições de Uptime...');
      await transitionsTab.click();
      await page.waitForTimeout(TIMINGS.auditTransitions);
    }

    // ══════════════════════════════════════════════════════════════════════
    // ENCERRAMENTO: NORMALIZAÇÃO & VOLTA AO DASHBOARD
    // ══════════════════════════════════════════════════════════════════════
    console.log('🔄 [Encerramento] Normalizando estado do sistema...');
    await triggerRecovery();
    await page.click('a[href="/"]');
    await page.waitForURL((url) => url.pathname === '/');
    await page.waitForTimeout(TIMINGS.closing);

    console.log('🎬 Gravação concluída. Finalizando sessão do navegador...');
  } finally {
    const video = page.video();
    await page.close();
    await context.close();
    await browser.close();

    if (serverProcess) {
      console.log('🛑 Encerrando servidor do modo DEMO...');
      serverProcess.kill('SIGTERM');
      await new Promise((r) => setTimeout(r, 1000));
    }

    if (video) {
      const rawWebmPath = await video.path();
      console.log(`📹 Vídeo bruto gravado: ${rawWebmPath}`);
      console.log('⚙️  Codificando MP4 profissional Full HD @ 60 FPS com FFmpeg...');

      execSync(
        `ffmpeg -y -i "${rawWebmPath}" -c:v libx264 -pix_fmt yuv420p -r 60 -crf 18 "${FINAL_MP4}"`,
        { stdio: 'inherit' }
      );

      // Remove webm temporário
      try {
        fs.unlinkSync(rawWebmPath);
      } catch (_) {}

      console.log('\n╔══════════════════════════════════════════════════════════╗');
      console.log('║  ✅ VÍDEO COMPLETO ABA POR ABA GERADO COM SUCESSO!       ║');
      console.log('╚══════════════════════════════════════════════════════════╝');
      console.log(`📁 Arquivo final: ${FINAL_MP4}\n`);
    }
  }
}

main().catch((err) => {
  console.error('❌ Falha na geração do vídeo de apresentação:', err);
  process.exit(1);
});
