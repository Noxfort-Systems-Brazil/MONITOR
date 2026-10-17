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
// File: scripts/demo/mock_telemetry.cjs
// Author: Gabriel Moraes
// Date: October 2026

const http = require('http');

const ENDPOINT = {
  hostname: '127.0.0.1',
  port: 22100,
  path: '/api/telemetry',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  }
};

function sendTelemetry(payload) {
  return new Promise((resolve, reject) => {
    const data = JSON.stringify(payload);
    const req = http.request(
      {
        ...ENDPOINT,
        headers: {
          ...ENDPOINT.headers,
          'Content-Length': Buffer.byteLength(data)
        }
      },
      (res) => {
        let resBody = '';
        res.on('data', (chunk) => { resBody += chunk; });
        res.on('end', () => {
          resolve({ statusCode: res.statusCode, body: resBody });
        });
      }
    );

    req.on('error', (err) => {
      reject(err);
    });

    req.write(data);
    req.end();
  });
}

/**
 * Envia batimentos cardíacos nominais para criar e manter dispositivos online.
 */
async function seedNormalTelemetry() {
  const normalEvents = [
    {
      origin: 'Synapse',
      level: 'INFO',
      category: 'SOFTWARE',
      message: 'Synapse Perception Gateway - Heartbeat OK (FPS: 60, Latency: 12ms)'
    },
    {
      origin: 'Carina',
      level: 'INFO',
      category: 'SOFTWARE',
      message: 'Carina Traffic Optimizer - Cycle Sync Heartbeat OK'
    },
    {
      origin: 'Sensor-Temp-42',
      level: 'INFO',
      category: 'HARDWARE',
      message: 'Painel Termo-Industrial Nominal: 24.5C - Heartbeat OK'
    },
    {
      origin: 'Substation-Energy-B',
      level: 'INFO',
      category: 'HARDWARE',
      message: 'Alimentação Trifásica 220V 60Hz Estável - Heartbeat OK'
    }
  ];

  for (const event of normalEvents) {
    try {
      await sendTelemetry(event);
    } catch (e) {
      console.warn(`[MOCK] Falha ao enviar telemetria para ${event.origin}:`, e.message);
    }
  }
}

/**
 * Dispara incidente crítico que altera o painel ao vivo.
 */
async function triggerCriticalIncident() {
  const incidentEvent = {
    origin: 'Synapse',
    level: 'CRITICAL',
    category: 'SOFTWARE',
    message: 'ALERTA CRÍTICO: Fila de inferência neural MARKVART excedeu 1250ms de latência!'
  };

  try {
    const res = await sendTelemetry(incidentEvent);
    console.log(`🚨 [MOCK] Incidente crítico injetado com sucesso (Status: ${res.statusCode})`);
    return res;
  } catch (e) {
    console.error('❌ [MOCK] Erro ao injetar incidente crítico:', e.message);
  }
}

/**
 * Dispara normalização do sistema.
 */
async function triggerRecovery() {
  const recoveryEvent = {
    origin: 'Synapse',
    level: 'INFO',
    category: 'SOFTWARE',
    message: 'Fila neural normalizada - Operação nominal restabelecida (System OK)'
  };

  try {
    const res = await sendTelemetry(recoveryEvent);
    console.log(`✅ [MOCK] Recuperação injetada com sucesso (Status: ${res.statusCode})`);
    return res;
  } catch (e) {
    console.error('❌ [MOCK] Erro ao injetar recuperação:', e.message);
  }
}

module.exports = {
  sendTelemetry,
  seedNormalTelemetry,
  triggerCriticalIncident,
  triggerRecovery
};
