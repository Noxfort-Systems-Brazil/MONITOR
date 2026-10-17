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
// File: internal/security/password_validator.go
// Author: Gabriel Moraes
// Date: October 2026

package security

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrPasswordTooShort  = errors.New("a senha deve conter no mínimo 8 caracteres")
	ErrPasswordNoNumber  = errors.New("a senha deve conter pelo menos um número (0-9)")
	ErrPasswordNoUpper   = errors.New("a senha deve conter pelo menos uma letra maiúscula (A-Z)")
	ErrPasswordNoLower   = errors.New("a senha deve conter pelo menos uma letra minúscula (a-z)")
	ErrPasswordNoSpecial = errors.New("a senha deve conter pelo menos um caractere especial (!@#$%...)")
)

const specialChars = "!@#$%^&*()-_=+[]{}|;:'\",.<>/?`~\\§¬¢£"

// ValidatePasswordComplexity enforces industrial security standards for user passwords:
// 1. Minimum 8 characters in length
// 2. At least one numeric digit (0-9)
// 3. At least one uppercase letter (A-Z)
// 4. At least one lowercase letter (a-z)
// 5. At least one special symbol / punctuation character
func ValidatePasswordComplexity(password string) error {
	if len([]rune(password)) < 8 {
		return ErrPasswordTooShort
	}

	var (
		hasDigit   bool
		hasUpper   bool
		hasLower   bool
		hasSpecial bool
	)

	for _, r := range password {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r) || strings.ContainsRune(specialChars, r):
			hasSpecial = true
		}
	}

	if !hasDigit {
		return ErrPasswordNoNumber
	}
	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}
