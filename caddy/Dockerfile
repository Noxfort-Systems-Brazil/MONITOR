# Multi-stage Dockerfile for Caddy with DuckDNS DNS-01 ACME plugin
# Noxfort Monitor™ - Copyright (C) 2026 Noxfort Systems

FROM caddy:builder-alpine AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/duckdns

FROM caddy:alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
