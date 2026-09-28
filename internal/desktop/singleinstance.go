// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
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
// File: internal/desktop/singleinstance.go
// Author: Gabriel Moraes
// Date: 2026-09-03

package desktop

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ErrAlreadyRunning indicates that another instance of Noxfort Monitor holds the system lock.
var ErrAlreadyRunning = errors.New("uma instância do Noxfort Monitor já está em execução")

func getLockFilePath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		return filepath.Join(runtimeDir, "noxfort-monitor.lock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("noxfort-monitor-%d.lock", os.Getuid()))
}

func getSocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		return filepath.Join(runtimeDir, "noxfort-monitor.sock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("noxfort-monitor-%d.sock", os.Getuid()))
}

// TryActivateExisting checks if another instance of Noxfort Monitor is already running.
// If an instance is active, it sends an activation command to bring the existing window to the front and returns true.
// If no instance is active, it returns false.
func TryActivateExisting() bool {
	sockPath := getSocketPath()
	conn, err := net.DialTimeout("unix", sockPath, 400*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("ACTIVATE\n"))
	return true
}

type singleInstanceLock struct {
	lockFile *os.File
	listener net.Listener
	sockPath string
	closeMu  sync.Mutex
	closed   bool
}

func (s *singleInstanceLock) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	var firstErr error
	if s.listener != nil {
		if err := s.listener.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	_ = os.Remove(s.sockPath)

	if s.lockFile != nil {
		_ = syscall.Flock(int(s.lockFile.Fd()), syscall.LOCK_UN)
		if err := s.lockFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// AcquireLockOrActivate guarantees that only a single instance of Noxfort Monitor runs on the host.
// It acquires an atomic kernel-level file lock (flock). If an instance is already active, it triggers
// window restoration via IPC and returns ErrAlreadyRunning.
func AcquireLockOrActivate(onActivate func()) (io.Closer, error) {
	lockPath := getLockFilePath()

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try acquiring an exclusive, non-blocking lock
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lockFile.Close()
		// Lock is held by another running process -> activate its window and signal ErrAlreadyRunning
		_ = TryActivateExisting()
		return nil, ErrAlreadyRunning
	}

	// Lock acquired: we are the primary active instance. Now start IPC activation server.
	sockPath := getSocketPath()
	_ = os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
		return nil, fmt.Errorf("failed to listen on single instance socket: %w", err)
	}

	instanceLock := &singleInstanceLock{
		lockFile: lockFile,
		listener: listener,
		sockPath: sockPath,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 64)
				n, _ := c.Read(buf)
				msg := strings.TrimSpace(string(buf[:n]))
				if msg == "ACTIVATE" || msg == "SHOW" {
					log.Println("[DESKTOP] Sinal de ativação recebido de outra instância. Restaurando janela...")
					if onActivate != nil {
						onActivate()
					}
				}
			}(conn)
		}
	}()

	return instanceLock, nil
}

// StartSingleInstanceServer starts listening for activation commands from subsequent invocations.
// Deprecated: prefer AcquireLockOrActivate for atomic kernel-level flock safety.
func StartSingleInstanceServer(onActivate func()) (io.Closer, error) {
	return AcquireLockOrActivate(onActivate)
}
