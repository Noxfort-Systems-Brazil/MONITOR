# 🔐 Segurança, Autenticação e Controle de Acesso (RBAC)

Este documento detalha a arquitetura de segurança do **Noxfort Monitor™ v2.0**, incluindo autenticação, ciclo de vida de tokens de sessão, hash de senhas com sal criptográfico e Controle de Acesso Baseado em Papéis (RBAC).

⬅️ [Central de Documentação](../README.md) | 🏛️ [Arquitetura](architecture.md) | 🔍 [Trilha de Auditoria](audit_trail.md) | 📡 [API](api_reference.md)

---

## 1. Arquitetura de Segurança e RBAC

O Noxfort Monitor implementa segregação rígida de privilégios:

* **`RoleAdmin` ("ADMIN")**: Privilégios totais de administração: cadastro de usuários, migração de bancos de dados, abertura de túnel reverso e configurações gerais do sistema.
* **`RoleOperator` ("OPERATOR")**: Privilégios operacionais: visualização do painel de telemetria, listagem de equipamentos e histórico de alertas.
* **Papéis de Roteamento de Notificações**:
  * **`TECHNICIAN`**: Recebe exclusivamente incidentes da categoria `HARDWARE`.
  * **`PROGRAMMER`**: Recebe exclusivamente incidentes da categoria `SOFTWARE`.
  * **`ADMIN`**: Recebe todas as categorias de incidentes.

---

## 2. Hash Criptográfico e Política de Senhas

### 2.1 Armazenamento Criptográfico e Sal Único
As senhas são protegidas por derivação SHA-256 com sal criptográfico em `internal/security/hasher.go`:
* Sal aleatório de 16 bytes gerado individualmente por usuário (`crypto/rand`).
* Senhas e hashes são estritamente excluídos da serialização JSON (`json:"-"`).
* Comparação em tempo constante para evitar ataques de temporização (*timing attacks*).

### 2.2 Política Industrial de Complexidade de Senhas (`password_validator.go`)
Para proteger sistemas críticos de automação contra ataques de força bruta, o validador [`internal/security/password_validator.go`](../../internal/security/password_validator.go) exige conformidade obrigatória via `ValidatePasswordComplexity()`:
* **Comprimento Mínimo**: No mínimo 8 caracteres.
* **Dígito Numérico**: Ao menos um número (`0-9`).
* **Letra Maiúscula**: Ao menos uma letra maiúscula (`A-Z`).
* **Letra Minúscula**: Ao menos uma letra minúscula (`a-z`).
* **Caractere Especial**: Ao menos um caractere especial (`!@#$%^&*()-_=+[]{}|;:'",.<>/?`~\\§¬¢£`).

Tentativas de cadastro ou criação de usuários sem esses critérios resultam em erro HTTP `400 Bad Request`.

### 2.3 Medidor Dinâmico de Força de Senha (`password_meter.js`)
As telas de login, primeiro acesso e gerenciamento de usuários incluem verificação em tempo real ([`web/static/js/password_meter.js`](../../web/static/js/password_meter.js)):
* Avaliação de entropia e pontuação de 0 a 4 conforme a digitação.
* Checklist interativo dos 5 requisitos de segurança com feedback visual imediato.
* Bloqueio preventivo de envio de formulário caso a senha seja vulnerável.

---

## 3. Ciclo de Vida da Sessão e AuthMiddleware

```mermaid
sequenceDiagram
    participant User as Operador / Navegador
    participant MW as AuthMiddleware
    participant SM as SessionManager
    participant App as Handler Protegido

    User->>MW: Requisição HTTP (Cookie: noxfort_session=XYZ)
    MW->>SM: ValidateSession("XYZ")
    alt Sessão Válida
        SM-->>MW: Session(User, RoleAdmin)
        MW->>App: Encaminha Requisição com Contexto do Usuário
        App-->>User: 200 OK (Renderiza Tela)
    else Sessão Inválida ou Expirada
        MW-->>User: Redireciona 303 para /login (ou 401 Unauthorized para /api/*)
    end
```

* **Armazenamento**: Mapa thread-safe em memória sincronizado por mutex.
* **Renovação Deslizante**: Interações ativas renovam automaticamente a janela de validade de 24 horas.
* **Intercepção Inteligente**: Requisições web sem sessão são redirecionadas com código HTTP `303 See Other`; chamadas para a API REST retornam JSON `401 Unauthorized`.

---

## 4. Hardening do Broker MQTT e Senhas PBKDF2

* **Desativação de Acesso Anônimo**: [`mosquitto.conf`](../../mosquitto/config/mosquitto.conf) define `allow_anonymous false`.
* **Criptografia Forte**: Credenciais geradas via `mosquitto_passwd` com hash PBKDF2 (SHA-512).
* **Setup Automatizado**: Comando `make broker-auth` gera credenciais e sincroniza automaticamente no `.env`.
* **Proteção Git**: Arquivos `passwd*` são ignorados no repositório.

---

## 5. Validação de Segurança de Ambiente (`env_validator.go`)

No boot do sistema, o validador inspeciona a integridade das configurações locais:
* **Restrição Automática de Permissões (`0600`)**: Em ambientes Linux, restringe o arquivo `.env` para leitura exclusiva pelo proprietário caso esteja acessível publicamente.
* **Detecção de Senhas Fracas**: Emite alerta crítico caso as credenciais padrão de superusuário (`admin` / `Admin@Noxfort2026!`) permaneçam ativas em produção.
* **Checagem de Credenciais MQTT**: Alerta caso `MQTT_USER` ou `MQTT_PASSWORD` estejam ausentes antes de iniciar o broker.

---

## 6. Bootstrap Idempotente do Superusuário (`superuser.go`)

Na inicialização, o sistema verifica as variáveis `MONITOR_ADMIN_USER` e `MONITOR_ADMIN_PASSWORD`:
* Se nenhuma conta de administrador existir, uma conta `ADMIN` é provisionada automaticamente.
* Caso credenciais customizadas sejam informadas em produção, a conta de teste padrão é purgada automaticamente para prevenir backdoors.

---

## 7. Prevenção de Vazamento de Segredos no CI/CD

O workflow automatizado do GitHub Actions ([`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)) audita todo commit para impedir o versionamento de arquivos `.env`, chaves privadas (`*.pem`, `*.key`) ou senhas do Mosquitto.
