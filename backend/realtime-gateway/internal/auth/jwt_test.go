package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewManager(t *testing.T) {
	secret := "test-secret"
	mgr := NewManager(secret)
	if mgr == nil {
		t.Fatal("expected manager, got nil")
	}
	if string(mgr.secret) != secret {
		t.Fatalf("expected secret %q, got %q", secret, string(mgr.secret))
	}
}

func TestManagerIssueAndValidate(t *testing.T) {
	mgr := NewManager("issue-validate-secret")

	token, exp, err := mgr.Issue()
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !exp.After(time.Now()) {
		t.Fatalf("expected expiration in future, got %v", exp)
	}

	if err := mgr.Validate(token); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestManagerValidateScenarios(t *testing.T) {
	goodMgr := NewManager("good-secret")

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	tests := []struct {
		name        string
		token       func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "expired token",
			token: func(t *testing.T) string {
				t.Helper()
				claims := Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
						Issuer:    "vibecat-gateway",
					},
				}
				tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				signed, signErr := tok.SignedString(goodMgr.secret)
				if signErr != nil {
					t.Fatalf("sign token: %v", signErr)
				}
				return signed
			},
			wantErr: true,
		},
		{
			name: "wrong secret",
			token: func(t *testing.T) string {
				t.Helper()
				otherMgr := NewManager("other-secret")
				signed, _, issueErr := otherMgr.Issue()
				if issueErr != nil {
					t.Fatalf("issue token: %v", issueErr)
				}
				return signed
			},
			wantErr: true,
		},
		{
			name: "wrong signing method",
			token: func(t *testing.T) string {
				t.Helper()
				claims := Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						IssuedAt:  jwt.NewNumericDate(time.Now()),
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						Issuer:    "vibecat-gateway",
					},
				}
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				signed, signErr := tok.SignedString(rsaKey)
				if signErr != nil {
					t.Fatalf("sign rsa token: %v", signErr)
				}
				return signed
			},
			wantErr:     true,
			errContains: "unexpected signing method",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := tc.token(t)
			err := goodMgr.Validate(tok)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tc.errContains != "" && (err == nil || !strings.Contains(err.Error(), tc.errContains)) {
				t.Fatalf("expected error containing %q, got %v", tc.errContains, err)
			}
		})
	}
}
