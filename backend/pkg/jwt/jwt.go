// Package jwt provides JWT token generation and verification for AMY MIS.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// Claims holds the custom JWT claims for AMY MIS tokens.
type Claims struct {
	UserID       uuid.UUID        `json:"user_id"`
	Phone        string           `json:"phone"`
	SystemRole   types.SystemRole `json:"system_role"`
	CrewMemberID *uuid.UUID       `json:"crew_member_id,omitempty"`
	SaccoID      *uuid.UUID       `json:"sacco_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair holds an access token and refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Manager handles JWT token operations.
type Manager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewManager creates a new JWT manager.
func NewManager(secret string, accessExpiryMinutes, refreshExpiryDays int) *Manager {
	return &Manager{
		secret:        []byte(secret),
		accessExpiry:  time.Duration(accessExpiryMinutes) * time.Minute,
		refreshExpiry: time.Duration(refreshExpiryDays) * 24 * time.Hour,
	}
}

// GenerateTokenPair creates a new access + refresh token pair.
func (m *Manager) GenerateTokenPair(userID uuid.UUID, phone string, role types.SystemRole, crewMemberID, saccoID *uuid.UUID) (*TokenPair, error) {
	accessToken, err := m.generateToken(userID, phone, role, crewMemberID, saccoID, m.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := m.generateToken(userID, phone, role, crewMemberID, saccoID, m.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// VerifyToken validates a JWT token and returns the claims.
func (m *Manager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (m *Manager) generateToken(userID uuid.UUID, phone string, role types.SystemRole, crewMemberID, saccoID *uuid.UUID, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Phone:        phone,
		SystemRole:   role,
		CrewMemberID: crewMemberID,
		SaccoID:      saccoID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "amy-mis",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}
