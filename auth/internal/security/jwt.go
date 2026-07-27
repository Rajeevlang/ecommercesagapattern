package security

import (
	"fmt"
	"time"

	"github.com/Rajeevlang/ecommercesagapattern/auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

func (m *JWTManager) GenerateToken(user *domain.User) (string, error) {
	claims := CustomClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secretKey, nil
		},
	)

	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return &domain.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
