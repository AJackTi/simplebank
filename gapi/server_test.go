package gapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	t.Helper()

	config := util.Config{
		TokenSymmetricKey:    util.RandomString(32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
	}

	server, err := NewServer(&config, store)
	require.NoError(t, err)

	return server
}

func newAuthContext(t *testing.T, server *Server, username string, tokenType token.TokenType) context.Context {
	t.Helper()

	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(authorizationHeader, newAuthHeader(t, server, username, tokenType)),
	)
}

func newAuthHeader(t *testing.T, server *Server, username string, tokenType token.TokenType) string {
	t.Helper()

	tokenValue, _, err := server.tokenMaker.CreateToken(username, tokenType, time.Minute)
	require.NoError(t, err)

	return fmt.Sprintf("bearer %s", tokenValue)
}
