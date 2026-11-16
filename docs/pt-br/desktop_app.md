# Noxfort Monitor — Aplicação Desktop e Modos de Execução

O Noxfort Monitor suporta execução como aplicativo desktop embarcado (Wails), servidor headless e modo de demonstração a partir do mesmo ponto de entrada em `cmd/server/main.go`.

---

## 1. Modos de Execução

* **Modo Desktop GUI (Padrão)**: Inicia o servidor HTTP/MQTT juntamente com a janela desktop Wails e ícone na bandeja do sistema (`internal/desktop/app.go`).
* **Modo Demonstração (`--demo`)**: Executa `RunDesktopServer()` diretamente sem interface Wails, otimizado para apresentações e gravações automatizadas (`make record-demo`).
* **Modo Headless (`--headless` / `--server-only`)**: Executa exclusivamente os serviços backend para ambientes Linux systemd, containers Docker e gateways industriais.

---

## 2. Recursos da Interface Desktop e Segurança

* **Polling ao Vivo do Dashboard**: A interface consulta `GET /api/dashboard/summary` a cada **10 segundos**, atualizando contadores e cartões em todos os 5 estados: `ONLINE`, `OFFLINE`, `PENDING`, `PAUSED` e `WAITING`.
* **Tela Dedicada de `403 Forbidden`**: Usuários `OPERATOR` (somente leitura) que tentam abrir rotas administrativas (ex.: `/users`) ou executar ações restritas visualizam a página `web/templates/forbidden.html` dentro da aplicação em vez de JSON bruto.
* **Troca de Senha pelo Próprio Usuário**: Todos os usuários autenticados — incluindo `OPERATOR` — podem alterar sua senha via `POST /api/users/password` (mediante `current_password`).
* **Redação de Credenciais e Chaves Únicas**: Chaves geradas na criação ou aprovação em `/devices` são exibidas uma única vez; carregamentos subsequentes mascaram `ApiKey`, `duckdns_token` e demais segredos no HTML (use `POST /devices/rotate-key` para rotação).

---

## 3. Encerramento Gracioso

* Ao fechar a janela desktop ou enviar `SIGINT`/`SIGTERM`, o contexto raiz cancela tarefas periódicas (`RetentionManager`, backups agendados, limpeza de sessões) e chama `AlertService.Shutdown(shutdownCtx)` para drenar notificações antes do fechamento do banco.
