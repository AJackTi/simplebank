package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AJackTi/simplebank/util"
)

func TestPasetoMaker(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, createdPayload, err := maker.CreateToken(username, AccessTokenType, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, createdPayload)

	payload, err := maker.VerifyToken(token, AccessTokenType)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.Equal(t, AccessTokenType, payload.TokenType)
	require.Equal(t, TokenIssuer, payload.Issuer)
	require.Equal(t, TokenAudience, payload.Audience)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, createdPayload, err := maker.CreateToken(util.RandomOwner(), AccessTokenType, -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, createdPayload)

	payload, err := maker.VerifyToken(token, AccessTokenType)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
	require.Empty(t, payload)
}
