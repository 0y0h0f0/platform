package xjwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager signs and validates JWT access tokens with a shared HMAC secret.
type Manager struct {
	secret []byte
}

// Claims extends registered JWT claims with the username needed by gateways and
// downstream services.
type Claims struct {
	jwtlib.RegisteredClaims
	Username string `json:"username"`
}

// NewManager creates a JWT manager for the provided shared secret.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Generate signs a token and returns its JTI so logout can blacklist the token.
func (m *Manager) Generate(userID, username string, ttl time.Duration) (token string, jti string, err error) {
	jti = uuid.NewString()
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
		},
		Username: username,
	}
	token, err = jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(m.secret)
	return
}

// Validate parses a token, checks the HMAC signing method and returns claims.
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(t *jwtlib.Token) (any, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
