package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-123456")
	code := m.Run()
	os.Unsetenv("JWT_SECRET")
	os.Exit(code)
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken("user-123", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify it parses back
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected userID=user-123, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role=admin, got %s", claims.Role)
	}
}

func TestGenerateToken_DifferentRoles(t *testing.T) {
	roles := []string{"owner", "admin", "dev", "viewer"}
	for _, role := range roles {
		token, err := GenerateToken("user-"+role, role)
		if err != nil {
			t.Fatalf("GenerateToken(%s) failed: %v", role, err)
		}
		claims, err := ParseToken(token)
		if err != nil {
			t.Fatalf("ParseToken for role %s failed: %v", role, err)
		}
		if claims.Role != role {
			t.Errorf("expected role=%s, got %s", role, claims.Role)
		}
	}
}

func TestParseToken_ValidToken(t *testing.T) {
	token, err := GenerateToken("user-abc", "dev")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != "user-abc" {
		t.Errorf("expected userID=user-abc, got %s", claims.UserID)
	}
	if claims.Role != "dev" {
		t.Errorf("expected role=dev, got %s", claims.Role)
	}
	if claims.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
	if claims.IssuedAt == nil {
		t.Error("expected IssuedAt to be set")
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	claims := &Claims{
		UserID: "user-expired",
		Role:   "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := getJWTSecret()
	if err != nil {
		t.Fatalf("failed to get JWT secret: %v", err)
	}
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = ParseToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseToken_InvalidSignature(t *testing.T) {
	claims := &Claims{
		UserID: "user-tampered",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret-key"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ParseToken(tokenString)
	if err == nil {
		t.Fatal("expected error for token with invalid signature")
	}
}

func TestParseToken_WrongSigningMethod(t *testing.T) {
	claims := &Claims{
		UserID: "user-wrong-method",
		Role:   "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	// Sign with RSA (none) instead of HMAC
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ParseToken(tokenString)
	if err == nil {
		t.Fatal("expected error for token with wrong signing method")
	}
}

func TestGenerateToken_CustomExpiry(t *testing.T) {
	// This tests that tokens generated with GenerateToken have a 24h expiry.
	// We verify by checking the ExpiresAt is roughly 24h from now.
	token, err := GenerateToken("user-custom", "dev")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	expectedExpiry := time.Now().Add(24 * time.Hour)
	expiresAtTime := *claims.ExpiresAt
	diff := expiresAtTime.Sub(expectedExpiry)
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Errorf("expected expiry ~24h from now, got %v (diff: %v)", expiresAtTime, diff)
	}
}

func TestGetJWTSecret(t *testing.T) {
	secret, err := getJWTSecret()
	if err != nil {
		t.Fatalf("getJWTSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Error("expected non-empty secret")
	}
}
