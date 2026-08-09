package auth

import (
	"context"
	"time"

	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

type SessionCreator interface {
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error)
}

type LoginResult struct {
	Session               db.Session
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func Login(
	ctx context.Context,
	store SessionCreator,
	tokenMaker token.Maker,
	config *util.Config,
	user db.User,
	userAgent string,
	clientIP string,
) (*LoginResult, error) {
	accessToken, accessPayload, err := tokenMaker.CreateToken(
		user.Username,
		token.AccessTokenType,
		config.AccessTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshPayload, err := tokenMaker.CreateToken(
		user.Username,
		token.RefreshTokenType,
		config.RefreshTokenDuration,
	)
	if err != nil {
		return nil, err
	}

	session, err := store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Session:               session,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
	}, nil
}
