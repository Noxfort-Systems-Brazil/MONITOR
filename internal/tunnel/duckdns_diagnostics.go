// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/tunnel/duckdns_diagnostics.go
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: Diagnostic testing and DNS resolution validation for DuckDNS.

package tunnel

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// Test verifies DuckDNS credentials and subdomain DNS resolution on demand.
func (d *DuckDNSDriver) Test(ctx context.Context, token, domain string) (*TestResult, error) {
	cleanToken := CleanDuckDNSToken(token)
	if cleanToken == "" {
		return nil, fmt.Errorf("duckdns token is required")
	}

	subdomain := CleanDuckDNSDomain(domain)
	if subdomain == "" {
		return nil, fmt.Errorf("duckdns subdomain is required")
	}

	if strings.Contains(subdomain, "ngrok") {
		return nil, fmt.Errorf("o domínio informado pertence ao Ngrok ('%s'). Por favor, insira o seu subdomínio do DuckDNS", subdomain)
	}

	ipv6 := DetectGlobalIPv6()

	// 1. Perform test update against DuckDNS API
	if err := d.executeUpdate(ctx, subdomain, cleanToken, ipv6); err != nil {
		return nil, err
	}

	// 2. Perform DNS resolution check
	fullDomain := fmt.Sprintf("%s.duckdns.org", subdomain)
	resolvedIPs, dnsErr := net.LookupHost(fullDomain)
	dnsMessage := ""
	if dnsErr != nil {
		dnsMessage = fmt.Sprintf("API DuckDNS validada com sucesso (Resposta: OK), mas a propagação do DNS ainda está em andamento (%v).", dnsErr)
	} else {
		dnsMessage = fmt.Sprintf("DuckDNS validado com sucesso! Resposta da API: OK. O subdomínio %s está apontando para: %s.", fullDomain, strings.Join(resolvedIPs, ", "))
	}

	return &TestResult{
		Success:     true,
		Message:     dnsMessage,
		Domain:      fullDomain,
		ResolvedIPs: resolvedIPs,
		IPv6Active:  ipv6 != "",
	}, nil
}
