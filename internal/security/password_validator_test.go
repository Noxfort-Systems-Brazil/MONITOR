// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Noxfort Systems
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
// File: internal/security/password_validator_test.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"errors"
	"testing"
)

func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectedErr error
	}{
		{
			name:        "Empty password",
			password:    "",
			expectedErr: ErrPasswordTooShort,
		},
		{
			name:        "Too short (less than 8 characters)",
			password:    "Ab1!def",
			expectedErr: ErrPasswordTooShort,
		},
		{
			name:        "Missing number",
			password:    "Abcdefgh!",
			expectedErr: ErrPasswordNoNumber,
		},
		{
			name:        "Missing uppercase letter",
			password:    "abcdefg1!",
			expectedErr: ErrPasswordNoUpper,
		},
		{
			name:        "Missing lowercase letter",
			password:    "ABCDEFG1!",
			expectedErr: ErrPasswordNoLower,
		},
		{
			name:        "Missing special character",
			password:    "Abcdefg123",
			expectedErr: ErrPasswordNoSpecial,
		},
		{
			name:        "Valid standard password",
			password:    "Admin@2026",
			expectedErr: nil,
		},
		{
			name:        "Valid password with 8 numbers, uppercase and special",
			password:    "12345678A!b",
			expectedErr: nil,
		},
		{
			name:        "Valid password with various special symbols",
			password:    "S3cur#Pass_2026",
			expectedErr: nil,
		},
		{
			name:        "Valid password with punctuation and symbols",
			password:    "Oper4tor.Nox$",
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordComplexity(tc.password)
			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("expected valid password, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.expectedErr)
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
				}
			}
		})
	}
}
