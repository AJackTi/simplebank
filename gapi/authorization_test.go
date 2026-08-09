package gapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func TestAuthorizeUserRejectsRefreshToken(t *testing.T) {
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	refreshToken, _, err := tokenMaker.CreateToken("alice", token.RefreshTokenType, time.Minute)
	require.NoError(t, err)

	server := &Server{tokenMaker: tokenMaker}
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(authorizationHeader, fmt.Sprintf("bearer %s", refreshToken)),
	)

	payload, err := server.authorizeUser(ctx)
	require.ErrorIs(t, err, token.ErrInvalidTokenType)
	require.Nil(t, payload)
}

func TestAuthorizeUserRejectsAmbiguousHeaders(t *testing.T) {
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	accessToken, _, err := tokenMaker.CreateToken("alice", token.AccessTokenType, time.Minute)
	require.NoError(t, err)

	server := &Server{tokenMaker: tokenMaker}
	testCases := []struct {
		name     string
		metadata metadata.MD
	}{
		{
			name: "ExtraField",
			metadata: metadata.Pairs(
				authorizationHeader,
				fmt.Sprintf("bearer %s extra", accessToken),
			),
		},
		{
			name: "MultipleHeaders",
			metadata: metadata.Pairs(
				authorizationHeader,
				fmt.Sprintf("bearer %s", accessToken),
				authorizationHeader,
				fmt.Sprintf("bearer %s", accessToken),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tc.metadata)

			payload, err := server.authorizeUser(ctx)
			require.Error(t, err)
			require.Nil(t, payload)
		})
	}
}
