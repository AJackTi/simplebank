package token

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractBearerToken(t *testing.T) {
	testCases := []struct {
		name      string
		headers   []string
		wantToken string
		wantErr   error
	}{
		{
			name:      "OK",
			headers:   []string{"bearer token-123"},
			wantToken: "token-123",
		},
		{
			name:    "MissingHeader",
			headers: nil,
			wantErr: ErrMissingAuthorizationHeader,
		},
		{
			name:    "MultipleHeaders",
			headers: []string{"bearer token-123", "bearer token-123"},
			wantErr: ErrMultipleAuthorizationHeaders,
		},
		{
			name:    "InvalidFormat",
			headers: []string{"bearer token-123 extra"},
			wantErr: ErrInvalidAuthorizationHeaderFormat,
		},
		{
			name:    "UnsupportedScheme",
			headers: []string{"basic token-123"},
			wantErr: ErrUnsupportedAuthorizationType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotToken, err := ExtractBearerToken(tc.headers)
			if tc.wantErr != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tc.wantErr))
				require.Empty(t, gotToken)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantToken, gotToken)
		})
	}
}
