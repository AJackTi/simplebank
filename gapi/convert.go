package gapi

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
)

func convertUser(user *db.User) *pb.User {
	return &pb.User{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		CreatedAt:         timestamppb.New(user.CreatedAt),
	}
}

func convertAccount(account *db.Account) *pb.Account {
	return &pb.Account{
		Id:        account.ID,
		Owner:     account.Owner,
		Balance:   account.Balance,
		Currency:  account.Currency,
		CreatedAt: timestamppb.New(account.CreatedAt),
	}
}

func convertAccounts(accounts []db.Account) []*pb.Account {
	result := make([]*pb.Account, 0, len(accounts))
	for i := range accounts {
		result = append(result, convertAccount(&accounts[i]))
	}
	return result
}

func convertEntry(entry *db.Entry) *pb.Entry {
	return &pb.Entry{
		Id:        entry.ID,
		AccountId: entry.AccountID,
		Amount:    entry.Amount,
		CreatedAt: timestamppb.New(entry.CreatedAt),
	}
}

func convertTransfer(transfer *db.Transfer) *pb.Transfer {
	return &pb.Transfer{
		Id:            transfer.ID,
		FromAccountId: transfer.FromAccountID,
		ToAccountId:   transfer.ToAccountID,
		Amount:        transfer.Amount,
		CreatedAt:     timestamppb.New(transfer.CreatedAt),
	}
}

func convertTransferTxResult(result *db.TransferTxResult) *pb.TransferTxResult {
	return &pb.TransferTxResult{
		Transfer:    convertTransfer(&result.Transfer),
		FromAccount: convertAccount(&result.FromAccount),
		ToAccount:   convertAccount(&result.ToAccount),
		FromEntry:   convertEntry(&result.FromEntry),
		ToEntry:     convertEntry(&result.ToEntry),
	}
}
