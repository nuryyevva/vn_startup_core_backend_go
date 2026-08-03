package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access tokens from refresh tokens so a refresh
// token can never be used to authenticate an API request and vice versa.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims is the JWT payload used for both access and refresh tokens.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrWrongType    = errors.New("wrong token type")
)

// TokenIssuer signs and parses JWTs using a single secret and configured TTLs.
type TokenIssuer struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewTokenIssuer(secret string, accessTTL, refreshTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

func (i *TokenIssuer) generate(userID uuid.UUID, role string, tokenType TokenType, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (i *TokenIssuer) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	return i.generate(userID, role, TokenTypeAccess, i.accessTokenTTL)
}

func (i *TokenIssuer) GenerateRefreshToken(userID uuid.UUID, role string) (string, error) {
	return i.generate(userID, role, TokenTypeRefresh, i.refreshTokenTTL)
}

// Parse validates the token signature and expiry and returns its claims.
func (i *TokenIssuer) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, t.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseAccessToken parses a token and rejects it unless it is an access token.
func (i *TokenIssuer) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims, err := i.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != TokenTypeAccess {
		return nil, ErrWrongType
	}
	return claims, nil
}

// ParseRefreshToken parses a token and rejects it unless it is a refresh token.
func (i *TokenIssuer) ParseRefreshToken(tokenStr string) (*Claims, error) {
	claims, err := i.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != TokenTypeRefresh {
		return nil, ErrWrongType
	}
	return claims, nil
}
