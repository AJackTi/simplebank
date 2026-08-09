package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/o1egl/paseto"
	"github.com/stretchr/testify/require"

	"github.com/AJackTi/simplebank/util"
)

func TestMakerRejectsUnexpectedTokenType(t *testing.T) {
	testCases := []struct {
		name     string
		newMaker func(t *testing.T) Maker
	}{
		{
			name: "JWT",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewJWTMaker(util.RandomString(32))
				require.NoError(t, err)
				return maker
			},
		},
		{
			name: "PASETO",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewPasetoMaker(util.RandomString(32))
				require.NoError(t, err)
				return maker
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker := tc.newMaker(t)

			accessToken, _, err := maker.CreateToken("alice", AccessTokenType, time.Minute)
			require.NoError(t, err)

			payload, err := maker.VerifyToken(accessToken, RefreshTokenType)
			require.ErrorIs(t, err, ErrInvalidTokenType)
			require.Nil(t, payload)

			refreshToken, _, err := maker.CreateToken("alice", RefreshTokenType, time.Minute)
			require.NoError(t, err)

			payload, err = maker.VerifyToken(refreshToken, AccessTokenType)
			require.ErrorIs(t, err, ErrInvalidTokenType)
			require.Nil(t, payload)
		})
	}
}

func TestMakerRejectsUnexpectedIssuerOrAudience(t *testing.T) {
	secretKey := util.RandomString(32)
	testCases := []struct {
		name      string
		newMaker  func(t *testing.T) Maker
		sealToken func(t *testing.T, payload *Payload) string
	}{
		{
			name: "JWT",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewJWTMaker(secretKey)
				require.NoError(t, err)
				return maker
			},
			sealToken: func(t *testing.T, payload *Payload) string {
				tokenValue, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(secretKey))
				require.NoError(t, err)
				return tokenValue
			},
		},
		{
			name: "PASETO",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewPasetoMaker(secretKey)
				require.NoError(t, err)
				return maker
			},
			sealToken: func(t *testing.T, payload *Payload) string {
				tokenValue, err := paseto.NewV2().Encrypt([]byte(secretKey), payload, nil)
				require.NoError(t, err)
				return tokenValue
			},
		},
	}

	claimCases := []struct {
		name   string
		mutate func(payload *Payload)
	}{
		{
			name: "Issuer",
			mutate: func(payload *Payload) {
				payload.Issuer = "another-service"
			},
		},
		{
			name: "Audience",
			mutate: func(payload *Payload) {
				payload.Audience = "another-api"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker := tc.newMaker(t)

			for _, claimCase := range claimCases {
				t.Run(claimCase.name, func(t *testing.T) {
					payload, err := NewPayload("alice", AccessTokenType, time.Minute)
					require.NoError(t, err)
					claimCase.mutate(payload)

					tokenValue := tc.sealToken(t, payload)
					verifiedPayload, err := maker.VerifyToken(tokenValue, AccessTokenType)
					require.ErrorIs(t, err, ErrInvalidToken)
					require.Nil(t, verifiedPayload)
				})
			}
		})
	}
}

func TestMakerRejectsLegacyTokensWithoutPurposeClaims(t *testing.T) {
	secretKey := util.RandomString(32)
	testCases := []struct {
		name      string
		newMaker  func(t *testing.T) Maker
		sealToken func(t *testing.T, payload *Payload) string
	}{
		{
			name: "JWT",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewJWTMaker(secretKey)
				require.NoError(t, err)
				return maker
			},
			sealToken: func(t *testing.T, payload *Payload) string {
				tokenValue, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(secretKey))
				require.NoError(t, err)
				return tokenValue
			},
		},
		{
			name: "PASETO",
			newMaker: func(t *testing.T) Maker {
				maker, err := NewPasetoMaker(secretKey)
				require.NoError(t, err)
				return maker
			},
			sealToken: func(t *testing.T, payload *Payload) string {
				tokenValue, err := paseto.NewV2().Encrypt([]byte(secretKey), payload, nil)
				require.NoError(t, err)
				return tokenValue
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker := tc.newMaker(t)
			payload, err := NewPayload("alice", AccessTokenType, time.Minute)
			require.NoError(t, err)
			payload.TokenType = ""
			payload.Issuer = ""
			payload.Audience = ""

			legacyToken := tc.sealToken(t, payload)
			verifiedPayload, err := maker.VerifyToken(legacyToken, AccessTokenType)
			require.ErrorIs(t, err, ErrInvalidToken)
			require.Nil(t, verifiedPayload)
		})
	}
}
