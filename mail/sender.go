package mail

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"strings"
	"time"
)

type EmailSender interface {
	SendEmail(ctx context.Context, to []string, subject string, content string) error
}

type SMTPSender struct {
	serverAddress string
	senderName    string
	senderAddress string
	auth          smtp.Auth
	sendMail      func(string, smtp.Auth, string, []string, []byte) error
}

func NewSMTPSender(serverAddress, senderName, senderAddress, username, password string) (*SMTPSender, error) {
	if strings.TrimSpace(serverAddress) == "" {
		return nil, fmt.Errorf("smtp server address is required")
	}
	if strings.TrimSpace(senderAddress) == "" {
		return nil, fmt.Errorf("sender address is required")
	}

	address, err := stdmail.ParseAddress(senderAddress)
	if err != nil {
		return nil, fmt.Errorf("parse sender address: %w", err)
	}

	sender := &SMTPSender{
		serverAddress: serverAddress,
		senderName:    senderName,
		senderAddress: address.Address,
		sendMail:      smtp.SendMail,
	}

	hasUsername := strings.TrimSpace(username) != ""
	hasPassword := strings.TrimSpace(password) != ""
	if hasUsername != hasPassword {
		return nil, fmt.Errorf("smtp username and password must be set together")
	}

	if hasUsername {
		host, _, err := net.SplitHostPort(serverAddress)
		if err != nil {
			return nil, fmt.Errorf("parse smtp server address: %w", err)
		}

		sender.auth = smtp.PlainAuth("", username, password, host)
	}

	return sender, nil
}

func (sender *SMTPSender) SendEmail(ctx context.Context, to []string, subject string, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	message, err := buildMessage(sender.senderName, sender.senderAddress, to, subject, content)
	if err != nil {
		return fmt.Errorf("build email message: %w", err)
	}

	return sender.sendMail(sender.serverAddress, sender.auth, sender.senderAddress, to, message)
}

func buildMessage(senderName, senderAddress string, to []string, subject string, content string) ([]byte, error) {
	var message bytes.Buffer

	fmt.Fprintf(&message, "From: %s\r\n", formatAddress(senderName, senderAddress))
	fmt.Fprintf(&message, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&message, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&message, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	message.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	message.WriteString("\r\n")

	quotedPrintableWriter := quotedprintable.NewWriter(&message)
	if _, err := quotedPrintableWriter.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := quotedPrintableWriter.Close(); err != nil {
		return nil, err
	}

	return message.Bytes(), nil
}

func formatAddress(senderName, senderAddress string) string {
	if strings.TrimSpace(senderName) == "" {
		return senderAddress
	}

	return (&stdmail.Address{
		Name:    senderName,
		Address: senderAddress,
	}).String()
}
