package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeySize = 32

// JWTMaker is a JSON Web Token maker
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}

	return &JWTMaker{secretKey}, nil
}

// CreateToken creates a new token for a specific username, type, and duration.
func (maker *JWTMaker) CreateToken(username string, tokenType TokenType, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, tokenType, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// VerifyToken checks if the token is valid and has the expected type.
func (maker *JWTMaker) VerifyToken(token string, expectedType TokenType) (*Payload, error) {
	keyFunc := func(parsedToken *jwt.Token) (interface{}, error) {
		if parsedToken.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}

		return []byte(maker.secretKey), nil
	}
	jwtToken, err := jwt.ParseWithClaims(
		token,
		&Payload{},
		keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}
	if err := payload.ValidFor(expectedType); err != nil {
		return nil, err
	}

	return payload, nil
}
