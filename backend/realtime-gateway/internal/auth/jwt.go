// Package auth provides JWT token generation and validation for the VibeCat gateway.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenDuration = 24 * time.Hour

// Claims holds the JWT payload.
type Claims struct {
	jwt.RegisteredClaims
}

// Manager handles JWT signing and validation.
type Manager struct {
	secret []byte
}

// NewManager creates a Manager with the given HMAC secret.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Issue creates a signed JWT token valid for 24 hours.
func (m *Manager) Issue() (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(tokenDuration)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "vibecat-gateway",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// Validate parses and validates a JWT token string.
// Returns an error if the token is invalid or expired.
func (m *Manager) Validate(tokenStr string) error {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return errors.New("token is not valid")
	}
	return nil
}
