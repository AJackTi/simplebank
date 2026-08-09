package gapi

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/token"
)

func (server *Server) RenewAccessToken(ctx context.Context, req *pb.RenewAccessTokenRequest) (*pb.RenewAccessTokenResponse, error) {
	if strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.GetRefreshToken(), token.RefreshTokenType)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %s", err)
	}

	if session.IsBlocked {
		return nil, status.Errorf(codes.Unauthenticated, "session is blocked")
	}
	if session.Username != refreshPayload.Username {
		return nil, status.Errorf(codes.Unauthenticated, "session username mismatch")
	}
	if session.RefreshToken != req.GetRefreshToken() {
		return nil, status.Errorf(codes.Unauthenticated, "session token mismatch")
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, status.Errorf(codes.Unauthenticated, "session expired")
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		token.AccessTokenType,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create access token")
	}

	return &pb.RenewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: timestamppb.New(accessPayload.ExpiredAt),
	}, nil
}
