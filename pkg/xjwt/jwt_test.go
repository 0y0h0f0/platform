package xjwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"task-platform/pkg/xjwt"
)

func TestGenerate(t *testing.T) {
	mgr := xjwt.NewManager("test-secret")
	token, jti, err := mgr.Generate("user-1", "alice", 2*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if jti == "" {
		t.Fatal("jti is empty")
	}
}

func TestValidate_Success(t *testing.T) {
	mgr := xjwt.NewManager("test-secret")
	token, _, err := mgr.Generate("user-1", "alice", 2*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := mgr.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("subject = %s, want user-1", claims.Subject)
	}
	if claims.Username != "alice" {
		t.Errorf("username = %s, want alice", claims.Username)
	}
}

func TestValidate_Expired(t *testing.T) {
	mgr := xjwt.NewManager("test-secret")
	token, _, err := mgr.Generate("user-1", "alice", -1*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = mgr.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	mgr := xjwt.NewManager("secret-a")
	token, _, err := mgr.Generate("user-1", "alice", 2*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	other := xjwt.NewManager("secret-b")
	_, err = other.Validate(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidate_WrongAlgorithm(t *testing.T) {
	mgr := xjwt.NewManager("test-secret")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	claims := jwtlib.RegisteredClaims{
		Subject:   uuid.NewString(),
		ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tokenString, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mgr.Validate(tokenString)
	if err == nil {
		t.Fatal("expected error for non-HMAC algorithm")
	}
}

func TestValidate_Tampered(t *testing.T) {
	mgr := xjwt.NewManager("test-secret")
	token, _, err := mgr.Generate("user-1", "alice", 2*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = mgr.Validate(token + "x")
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}
