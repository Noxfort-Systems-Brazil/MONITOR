# Noxfort Monitor — Arquitetura e Ciclo de Vida do Banco de Dados

O Noxfort Monitor utiliza uma camada de persistência de motor duplo capaz de alternar em tempo real, sem reinicialização, entre **SQLite (modo WAL)** e **PostgreSQL**, com migração transacional, backups quentes, restauração verificada e retenção automática de dados.

---

## 1. Motor Duplo e Recarregamento Dinâmico

```
+---------------------------------------------------------------+
|                         DBManager                             |
|   - Gerencia *sql.DB ativo + driver ("sqlite" / "postgres")   |
|   - Coordena MigrateData + Switch()                           |
+------------------------------+--------------------------------+
                               | SetDB(db *sql.DB, driver string)
                               v
   +--------------------------------------------------------+
   | ReloadableRepository (7 Repositórios)                  |
   |  1. DeviceRepository     5. AuditRepository            |
   |  2. TelemetryRepository  6. UserRepository             |
   |  3. SettingsRepository   7. DBSessionStore             |
   |  4. ContactRepository                                  |
   +--------------------------------------------------------+
```

* **SQLite em Modo WAL**: Inicializado com `journal_mode=WAL`, `synchronous=NORMAL` e `busy_timeout=5000` para suportar leituras concorrentes e alta taxa de escrita sem bloqueios.
* **Paridade de Esquema (`9` Tabelas, `12` Índices)**: Ambos os dialetos mantêm estrutura idêntica nas tabelas `devices` (incluindo `api_key`, `status`, `expires_at`), `telemetry` (coluna `device_id`), `settings`, `contacts`, `users`, `sessions` (`token_hash`, `expires_at`), `security_audit_logs`, `alert_dispatch_logs` e `device_state_transitions`.
* **Troca em Tempo Real**: `SetDB(db *sql.DB, driver string)` rebinda todos os 7 repositórios `ReloadableRepository` sem derrubar os servidores HTTP ou MQTT.

---

## 2. Migração Transacional ao Vivo (`MigrateData`)

* Durante a transição SQLite ↔ PostgreSQL, `Migrator.MigrateData` copia todas as **9 tabelas** dentro de uma **única transação SQL**.
* Caso ocorra qualquer erro em alguma tabela, a transação sofre rollback integral e `DBManager.Switch` é abortado com segurança, impedindo estados parciais (*split-brain*).

---

## 3. Backups Quentes e Restauração Verificada (`BackupManager`)

* **Snapshots sem Bloqueio**:
  * **SQLite**: Executa `VACUUM INTO` para gerar cópia consistente e compactada sem interromper a telemetria.
  * **PostgreSQL**: Invoca `pg_dump -F c` em formato customizado.
* **Agendamento Automático**: `BackupManager.StartScheduledBackups` executa backups periódicos conforme `MONITOR_BACKUP_INTERVAL_HOURS` (padrão `24h`), retendo até **`7`** cópias recentes.
* **Download e Restauração Segura**:
  * `GET /api/settings/database/backups/download` valida caminhos via `ResolveBackupPath` contra *path traversal*.
  * `POST /api/settings/database/restore` valida SQLite com `PRAGMA quick_check` antes de substituir via `RestoreSQLiteFile`, ou utiliza `pg_restore` no PostgreSQL.

---

## 4. Retenção Automática de Histórico (`RetentionManager`)

* Configurado via `MONITOR_RETENTION_DAYS` (padrão **`90` dias**), com varredura a cada **`24` horas**.
* Remove registros históricos expirados de 4 tabelas de alto volume, preservando cadastros de dispositivos, usuários, sessões ativas e configurações:
  1. `telemetry`
  2. `alert_dispatch_logs`
  3. `security_audit_logs`
  4. `device_state_transitions`
