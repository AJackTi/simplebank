package gapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	mockdb "github.com/AJackTi/simplebank/db/mock"
	db "github.com/AJackTi/simplebank/db/sqlc"
	"github.com/AJackTi/simplebank/pb"
	"github.com/AJackTi/simplebank/token"
	"github.com/AJackTi/simplebank/util"
)

func TestGatewayCreateAccount(t *testing.T) {
	user := util.RandomOwner()
	account := randomAccount(user)
	account.Currency = util.USD

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().
		CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
			Owner:    user,
			Balance:  0,
			Currency: util.USD,
		})).
		Return(account, nil)

	grpcMux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}))
	err := pb.RegisterSimpleBankHandlerServer(context.Background(), grpcMux, server)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/accounts", strings.NewReader(`{"currency":"USD"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", newAuthHeader(t, server, user, token.AccessTokenType))

	grpcMux.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var rsp pb.CreateAccountResponse
	require.NoError(t, protojson.Unmarshal(recorder.Body.Bytes(), &rsp))
	require.True(t, proto.Equal(convertAccount(&account), rsp.Account))
}
