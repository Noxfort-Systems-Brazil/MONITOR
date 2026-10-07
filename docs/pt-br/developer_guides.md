# 👨‍💻 Guia Avançado do Desenvolvedor: Noxfort Monitor™

Este documento fornece diretrizes de engenharia de software para compilação, manutenção e extensão do **Noxfort Monitor™ v2.0**, cobrindo configuração do ambiente local, convenções de arquitetura **SOLID**, padrão de repositórios com **hot-reload**, injeção de dependências e modelo de concorrência com goroutines.

⬅️ [Central de Documentação](README.md) | 🏛️ [Arquitetura](architecture.md) | 🧪 [Testes e QA](testing.md) | 🗄️ [Banco de Dados](database.md)

---

## 1. Ambiente Local e Compilação

### 1.1 Pré-requisitos do Sistema (Ubuntu / Debian)
* **Go Toolkit**: Versão 1.22 ou superior.
* **Bibliotecas C/GTK e WebKit**: Necessárias para o runtime desktop Wails v2 e a bandeja do sistema (systray):
  ```bash
  sudo apt-get update
  sudo apt-get install -y \
      libgtk-3-dev \
      libwebkit2gtk-4.1-dev \
      libappindicator3-dev || sudo apt-get install -y libayatana-appindicator3-dev
  ```
* **Broker Mosquitto Local**:
  ```bash
  make broker-install
  make broker-auth
  make broker-start
  ```

### 1.2 Comandos Essenciais do Makefile

| Comando | Descrição |
| :--- | :--- |
| `make build` | Compila o binário desktop com tags de produção (`bin/noxfort-monitor`). |
| `make run` | Executa a aplicação completa em modo de desenvolvimento com interface gráfica. |
| `make run-headless` | Executa o servidor em modo Headless (sem abrir janelas gráficas). |
| `make test` | Executa a suíte completa de testes automatizados (backend Go + frontend Vitest). |
| `make deb` | Gera o pacote Debian de distribuição (`noxfort-monitor_2.0.1_amd64.deb`). |
| `make broker-start` | Inicia o broker MQTT nativo (Mosquitto). |
| `make broker-stop` | Para o broker MQTT. |
| `make broker-status`| Verifica o status de execução do Mosquitto. |

---

## 2. Padrões de Projeto e Arquitetura

O Noxfort Monitor implementa rigorosamente os princípios **SOLID** dentro de uma arquitetura orientada a eventos.

### 2.1 Raiz de Composição (Composition Root)
Toda instanciação de objetos e amarração de dependências ocorre explicitamente em [`cmd/server/main.go`](../../cmd/server/main.go).
* **Zero Variáveis Globais**: Nenhum pacote mantém estado global mutável ou rotinas mágicas de `init()`.
* **Injeção de Dependências (DI)**: Construtores recebem interfaces, garantindo desacoplamento e facilitando testes unitários com mocks em memória.

### 2.2 Padrão Repository com Recarregamento a Quente (Hot-Reload)
A camada de persistência em [`internal/storage`](../../internal/storage) isola os mecanismos de armazenamento através das interfaces definidas em [`internal/domain`](../../internal/domain):
* Para suportar a troca dinâmica entre SQLite e PostgreSQL em tempo de execução sem reiniciar a aplicação, os repositórios implementam a interface `ReloadableRepository`:
  ```go
  type ReloadableRepository interface {
      SetDB(db *sql.DB, driver string)
  }
  ```
* Quando o operador alterna entre SQLite e PostgreSQL, o [`DBManager`](../../internal/storage/db_manager.go) notifica todos os repositórios registrados via `SetDB`.

### 2.3 Boas Práticas de Concorrência
* **Rastreamento de Estado pelo Watchdog**: A presença dos equipamentos é rastreada em memória via [`SystemStatusTracker`](../../internal/monitor/tracker.go), protegida por um `sync.RWMutex` para eliminar race conditions.
* **Disparo Assíncrono de Alertas**: Notificações disparadas por E-mail (SMTP) e Telegram operam em goroutines independentes dentro de [`AlertService.TriggerAlert()`](../../internal/monitor/alerts.go). A latência de rede dos servidores externos nunca bloqueia o pipeline de ingestão MQTT.

---

## 3. Estendendo o Sistema

### 3.1 Adicionando um Novo Canal de Notificação (ex: Discord, Slack ou WhatsApp)
1. Implemente a interface `NotificationChannel` definida em [`internal/monitor/channel.go`](../../internal/monitor/channel.go):
   ```go
   type NotificationChannel interface {
       Name() string
       Send(settings *domain.Settings, contact *domain.Contact, identifier string, event *domain.IncomingEvent) error
       Recipient(contact *domain.Contact) string
   }
   ```
2. Adicione os campos de configuração necessários na entidade [`Settings`](../../internal/domain/settings.go).
3. Conecte o novo canal no construtor do `AlertService` em `cmd/server/main.go`.

### 3.2 Adicionando uma Nova Tabela ou Entidade
1. Defina a struct e a interface do repositório em `internal/domain`.
2. Implemente o repositório em `internal/storage`, implementando `ReloadableRepository` e utilizando [`storage.AdaptQuery`](../../internal/storage/query_adapter.go) para suportar dialetos SQLite e PostgreSQL simultaneamente.
3. Adicione o DDL correspondente no `NewDatabase` em `database.go` (para SQLite) e no `InitPostgresSchema` em `postgres_schema.go` (para PostgreSQL).
4. Caso a entidade contenha dados que devam ser preservados na migração de bancos, adicione a rotina de cópia em `migrator.go`.
