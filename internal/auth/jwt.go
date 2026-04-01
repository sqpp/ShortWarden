package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func (m *JWTManager) Mint(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.ttl)
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(m.secret)
	return s, exp, err
}

func (m *JWTManager) Parse(tokenString string) (uuid.UUID, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	tok, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return uuid.UUID{}, errors.New("invalid token claims")
	}
	return uuid.Parse(claims.UserID)
}

