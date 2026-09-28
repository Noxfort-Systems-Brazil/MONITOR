// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/tunnel_handler_mock_test.go
// Author: Gabriel Moraes
// Date: 2026-09-14
// Description: Mock implementation of tunnel.Service for HTTP transport tests.

package http

import (
	"context"
	"sync"

	"noxfort-monitor-server/internal/tunnel"
)

type mockTunnelService struct {
	mu           sync.Mutex
	status       tunnel.Status
	startErr     error
	stopErr      error
	startedToken string
	startedDom   string
	isAvailable  bool
	testResult   *tunnel.TestResult
	testErr      error
	testedToken  string
	testedDom    string
}

func (m *mockTunnelService) Start(authToken, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedToken = authToken
	m.startedDom = domain
	if m.startErr != nil {
		return m.startErr
	}
	m.status.State = tunnel.StateOnline
	m.status.PublicURL = "https://" + domain
	m.status.TelemetryURL = "https://" + domain + "/api/telemetry"
	return nil
}

func (m *mockTunnelService) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	m.status.State = tunnel.StateOffline
	m.status.PublicURL = ""
	m.status.TelemetryURL = ""
	return nil
}

func (m *mockTunnelService) GetStatus() tunnel.Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *mockTunnelService) IsBinaryAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isAvailable
}

func (m *mockTunnelService) TestConnection(ctx context.Context, token, domain string) (*tunnel.TestResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testedToken = token
	m.testedDom = domain
	if m.testErr != nil {
		return nil, m.testErr
	}
	if m.testResult != nil {
		return m.testResult, nil
	}
	return &tunnel.TestResult{
		Success:     true,
		Message:     "Mock validation success",
		Domain:      domain + ".duckdns.org",
		ResolvedIPs: []string{"1.2.3.4"},
	}, nil
}
