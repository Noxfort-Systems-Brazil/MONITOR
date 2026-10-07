# 🛡️ Proxy Inverse Caddy & DuckDNS (HTTPS Automatique) : Noxfort Monitor™

Ce document décrit l'architecture, le déploiement et l'exploitation de **Caddy Server** intégré avec **DuckDNS** dans **Noxfort Monitor™ v2.0**. Cette topologie assure une terminaison TLS/SSL automatique de niveau entreprise via le **défi ACME DNS-01**, éliminant les flux non chiffrés sur le réseau étendu (WAN) et exposant les ports standards (`80`, `443` et `WSS`).

⬅️ [Centre de Documentation](README.md) | 🌐 [Accès Distant](remote_access.md) | 🚀 [Déploiement](deployment.md) | 📡 [Référence API](api_reference.md)

---

## 1. Vue d'Ensemble de l'Architecture

```mermaid
graph LR
    subgraph "Internet Public / WAN"
        Carina[Agent Carina]
        Synapse[Agent Synapse]
        Browser[Navigateur Opérateur]
    end

    subgraph "Passerelle Caddy Edge (Docker)"
        Caddy[Serveur Caddy :80/:443]
        LetEncrypt[Let's Encrypt / ZeroSSL]
        DuckAPI[API DNS-01 DuckDNS]
    end

    subgraph "Services Locaux sur l'Hôte"
        Monitor[Noxfort Monitor Core :22100]
        Mosquitto[Mosquitto MQTT :9001 WSS]
    end

    Carina -->|HTTPS POST :443| Caddy
    Synapse -->|HTTPS POST :443| Caddy
    Browser -->|HTTPS GET :443| Caddy

    Caddy -.->|Défi DNS-01 ACME| DuckAPI
    DuckAPI -.->|Validation du Domaine| LetEncrypt

    Caddy -->|Proxy Inverse HTTP| Monitor
    Caddy -->|Proxy WSS /mqtt| Mosquitto
```

### Avantages Clés :
1. **HTTPS Zéro Configuration** : Caddy émet et renouvelle automatiquement des certificats valides.
2. **Défi ACME DNS-01** : Ne requiert pas l'ouverture du port 80 entrant.
3. **WebSockets Sécurisés (WSS)** : MQTT-over-WebSockets chiffré via `wss://votre-domaine.duckdns.org/mqtt`.
4. **Standardisation des Ports** : Centralise le trafic sur le port 443 sans exposer `:22100`.

---

## 2. Déploiement

```bash
make caddy-build
make services-start
make services-logs
make services-stop
```
