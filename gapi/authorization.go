package gapi

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"

	"github.com/AJackTi/simplebank/token"
)

const (
	authorizationHeader = "authorization"
)

func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	accessToken, err := token.ExtractBearerToken(md.Get(authorizationHeader))
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	payload, err := server.tokenMaker.VerifyToken(accessToken, token.AccessTokenType)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	return payload, nil
}
