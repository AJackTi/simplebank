package gapi

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/val"
)

func (server *Server) CreateTransfer(ctx context.Context, req *pb.CreateTransferRequest) (*pb.CreateTransferResponse, error) {
	violations := validateCreateTransferRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	fromAccount, err := server.store.GetAccount(ctx, req.GetFromAccountId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "source account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get source account: %s", err)
	}
	if fromAccount.Owner != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "source account does not belong to the authenticated user")
	}
	if fromAccount.Currency != req.GetCurrency() {
		return nil, status.Errorf(codes.InvalidArgument, "source account currency does not match transfer currency")
	}

	toAccount, err := server.store.GetAccount(ctx, req.GetToAccountId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "destination account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get destination account: %s", err)
	}
	if toAccount.Currency != req.GetCurrency() {
		return nil, status.Errorf(codes.InvalidArgument, "destination account currency does not match transfer currency")
	}

	result, err := server.store.TransferTx(ctx, db.TransferTxParams{
		FromAccountID: req.GetFromAccountId(),
		ToAccountID:   req.GetToAccountId(),
		Amount:        req.GetAmount(),
	})
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidTransferAmount),
			errors.Is(err, db.ErrSameAccountTransfer),
			errors.Is(err, db.ErrCurrencyMismatch):
			return nil, status.Errorf(codes.InvalidArgument, "invalid transfer: %s", err)
		case errors.Is(err, db.ErrInsufficientFunds):
			return nil, status.Errorf(codes.ResourceExhausted, "insufficient funds: %s", err)
		case errors.Is(err, sql.ErrNoRows):
			return nil, status.Errorf(codes.NotFound, "account not found")
		default:
			return nil, status.Errorf(codes.Internal, "failed to create transfer: %s", err)
		}
	}

	return &pb.CreateTransferResponse{Result: convertTransferTxResult(result)}, nil
}

func validateCreateTransferRequest(req *pb.CreateTransferRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateAccountID(req.GetFromAccountId()); err != nil {
		violations = append(violations, fieldViolation("from_account_id", err))
	}
	if err := val.ValidateAccountID(req.GetToAccountId()); err != nil {
		violations = append(violations, fieldViolation("to_account_id", err))
	}
	if err := val.ValidateTransferAmount(req.GetAmount()); err != nil {
		violations = append(violations, fieldViolation("amount", err))
	}
	if err := val.ValidateCurrency(req.GetCurrency()); err != nil {
		violations = append(violations, fieldViolation("currency", err))
	}
	return
}
