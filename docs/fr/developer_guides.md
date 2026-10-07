# 👨‍💻 Guide Développeur Avancé : Noxfort Monitor™

Ce document fournit les directives d'ingénierie logicielle pour compiler, maintenir et étendre **Noxfort Monitor™ v2.0** : configuration locale, architecture **SOLID**, dépôts avec rechargement à chaud (**hot-reload**), injection de dépendances et concurrence Go.

⬅️ [Centre de Documentation](README.md) | 🏛️ [Architecture](architecture.md) | 🧪 [Tests & QA](testing.md) | 🗄️ [Base de Données](database.md)

---

## 1. Environnement Local & Commandes

| Commande | Description |
| :--- | :--- |
| `make build` | Compile le binaire de bureau avec balises de production (`bin/noxfort-monitor`). |
| `make run` | Exécute l'application en mode développement avec interface graphique. |
| `make run-headless` | Exécute le serveur en mode Headless (sans interface graphique). |
| `make test` | Exécute l'ensemble des tests automatisés (Go + Vitest). |
| `make deb` | Génère le paquet Debian (`noxfort-monitor_2.0.1_amd64.deb`). |
| `make install-deb` | Installe ou met à jour le paquet `.deb` généré via `apt`. |
| `make uninstall-deb`| Désinstalle le paquet `noxfort-monitor` du système. |
| `make broker-start` | Démarre le broker MQTT natif (Mosquitto). |
| `make broker-stop` | Arrête le broker MQTT. |
