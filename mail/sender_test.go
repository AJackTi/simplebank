package mail

import (
	"context"
	"errors"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSMTPSenderBuildsAuthWhenCredentialsAreProvided(t *testing.T) {
	sender, err := NewSMTPSender(
		"smtp.example.com:587",
		"SimpleBank",
		"noreply@simplebank.dev",
		"username",
		"password",
	)
	require.NoError(t, err)
	require.NotNil(t, sender.auth)
}

func TestNewSMTPSenderRejectsPartialCredentials(t *testing.T) {
	_, err := NewSMTPSender(
		"smtp.example.com:587",
		"SimpleBank",
		"noreply@simplebank.dev",
		"username",
		"",
	)
	require.Error(t, err)
}

func TestSMTPSenderSendEmailBuildsExpectedMessage(t *testing.T) {
	sender, err := NewSMTPSender(
		"localhost:1025",
		"SimpleBank",
		"noreply@simplebank.dev",
		"",
		"",
	)
	require.NoError(t, err)

	var gotAddr string
	var gotAuth smtp.Auth
	var gotFrom string
	var gotTo []string
	var gotMsg []byte
	sender.sendMail = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr = addr
		gotAuth = auth
		gotFrom = from
		gotTo = append([]string(nil), to...)
		gotMsg = append([]byte(nil), msg...)
		return nil
	}

	err = sender.SendEmail(context.Background(), []string{"user@example.com"}, "Verify your SimpleBank account", "hello")
	require.NoError(t, err)
	require.Equal(t, "localhost:1025", gotAddr)
	require.Nil(t, gotAuth)
	require.Equal(t, "noreply@simplebank.dev", gotFrom)
	require.Equal(t, []string{"user@example.com"}, gotTo)
	require.Contains(t, string(gotMsg), "From: \"SimpleBank\" <noreply@simplebank.dev>")
	require.Contains(t, string(gotMsg), "To: user@example.com")
	require.Contains(t, string(gotMsg), "Subject: Verify your SimpleBank account")
	require.Contains(t, string(gotMsg), "Content-Type: text/plain; charset=\"UTF-8\"")
	require.Contains(t, string(gotMsg), "\r\n\r\nhello")
}

func TestSMTPSenderSendEmailRejectsEmptyRecipients(t *testing.T) {
	sender, err := NewSMTPSender(
		"localhost:1025",
		"SimpleBank",
		"noreply@simplebank.dev",
		"",
		"",
	)
	require.NoError(t, err)

	err = sender.SendEmail(context.Background(), nil, "subject", "body")
	require.Error(t, err)
}

func TestSMTPSenderSendEmailReturnsContextError(t *testing.T) {
	sender, err := NewSMTPSender(
		"localhost:1025",
		"SimpleBank",
		"noreply@simplebank.dev",
		"",
		"",
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = sender.SendEmail(ctx, []string{"user@example.com"}, "subject", "body")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSMTPSenderSendEmailReturnsSendError(t *testing.T) {
	sender, err := NewSMTPSender(
		"localhost:1025",
		"SimpleBank",
		"noreply@simplebank.dev",
		"",
		"",
	)
	require.NoError(t, err)

	sender.sendMail = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return errors.New("smtp down")
	}

	err = sender.SendEmail(context.Background(), []string{"user@example.com"}, "subject", "body")
	require.Error(t, err)
	require.Contains(t, err.Error(), "smtp down")
}
