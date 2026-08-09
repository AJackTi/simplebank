package gapi

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/val"
)

func (server *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	violations := validateCreateAccountRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	account, err := server.store.CreateAccount(ctx, db.CreateAccountParams{
		Owner:    authPayload.Username,
		Balance:  0,
		Currency: req.GetCurrency(),
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "account already exists: %s", err)
			case "foreign_key_violation":
				return nil, status.Errorf(codes.FailedPrecondition, "user does not exist: %s", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %s", err)
	}

	return &pb.CreateAccountResponse{Account: convertAccount(&account)}, nil
}

func (server *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	violations := validateGetAccountRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	account, err := server.store.GetAccount(ctx, req.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get account: %s", err)
	}

	if account.Owner != authPayload.Username {
		return nil, status.Errorf(codes.PermissionDenied, "account does not belong to the authenticated user")
	}

	return &pb.GetAccountResponse{Account: convertAccount(&account)}, nil
}

func (server *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	violations := validateListAccountsRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	accounts, err := server.store.ListAccounts(ctx, db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.GetPageSize(),
		Offset: (req.GetPageId() - 1) * req.GetPageSize(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %s", err)
	}

	return &pb.ListAccountsResponse{Accounts: convertAccounts(accounts)}, nil
}

func validateCreateAccountRequest(req *pb.CreateAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateCurrency(req.GetCurrency()); err != nil {
		violations = append(violations, fieldViolation("currency", err))
	}
	return
}

func validateGetAccountRequest(req *pb.GetAccountRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateAccountID(req.GetId()); err != nil {
		violations = append(violations, fieldViolation("id", err))
	}
	return
}

func validateListAccountsRequest(req *pb.ListAccountsRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidatePageID(req.GetPageId()); err != nil {
		violations = append(violations, fieldViolation("page_id", err))
	}
	if err := val.ValidatePageSize(req.GetPageSize()); err != nil {
		violations = append(violations, fieldViolation("page_size", err))
	}
	return
}
