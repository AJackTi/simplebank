package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Different types of error returned by the VerifyToken function
var (
	ErrInvalidToken     = errors.New("token is invalid")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("token has invalid type")
)

// TokenType identifies the purpose for which a token may be used.
type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"

	TokenIssuer   = "simplebank"
	TokenAudience = "simplebank-api"
)

// Payload contains the payload data of the token
type Payload struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	TokenType TokenType `json:"token_type"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// NewPayload creates a new token payload with a specific username, type, and duration.
func NewPayload(username string, tokenType TokenType, duration time.Duration) (*Payload, error) {
	if !tokenType.valid() {
		return nil, ErrInvalidTokenType
	}

	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Payload{
		ID:        tokenID,
		Username:  username,
		TokenType: tokenType,
		Issuer:    TokenIssuer,
		Audience:  TokenAudience,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}, nil
}

// Valid checks if the token payload is valid or not
func (payload *Payload) Valid() error {
	if !payload.TokenType.valid() || payload.Issuer != TokenIssuer || payload.Audience != TokenAudience {
		return ErrInvalidToken
	}

	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}

	return nil
}

// ValidFor checks that the payload is valid for the expected token purpose.
func (payload *Payload) ValidFor(expectedType TokenType) error {
	if err := payload.Valid(); err != nil {
		return err
	}

	if !expectedType.valid() || payload.TokenType != expectedType {
		return ErrInvalidTokenType
	}

	return nil
}

func (tokenType TokenType) valid() bool {
	return tokenType == AccessTokenType || tokenType == RefreshTokenType
}

func (payload *Payload) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.ExpiredAt), nil
}

func (payload *Payload) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(payload.IssuedAt), nil
}

func (payload *Payload) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (payload *Payload) GetIssuer() (string, error) {
	return payload.Issuer, nil
}

func (payload *Payload) GetSubject() (string, error) {
	return payload.Username, nil
}

func (payload *Payload) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{payload.Audience}, nil
}
