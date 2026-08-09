package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigFromEnvironmentWithoutFile(t *testing.T) {
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_SOURCE", "postgresql://user:password@localhost:55432/simple_bank?sslmode=disable")
	t.Setenv("MIGRATION_URL", "file://db/migration")
	t.Setenv("REDIS_ADDRESS", "localhost:56379")
	t.Setenv("SMTP_SERVER_ADDRESS", "localhost:1025")
	t.Setenv("EMAIL_SENDER_NAME", "SimpleBank")
	t.Setenv("EMAIL_SENDER_ADDRESS", "noreply@simplebank.dev")
	t.Setenv("HTTP_SERVER_ADDRESS", "127.0.0.1:58080")
	t.Setenv("GRPC_SERVER_ADDRESS", "127.0.0.1:59090")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("ACCESS_TOKEN_DURATION", "15m")
	t.Setenv("REFRESH_TOKEN_DURATION", "24h")

	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, "test", config.Environment)
	require.Equal(t, "postgres", config.DBDriver)
	require.Equal(t, 15*time.Minute, config.AccessTokenDuration)
	require.Equal(t, 24*time.Hour, config.RefreshTokenDuration)
}

func TestConfigValidateRejectsWeakTokenKey(t *testing.T) {
	config := Config{
		Environment:          "test",
		DBDriver:             "postgres",
		DBSource:             "postgresql://user:password@localhost:55432/simple_bank?sslmode=disable",
		MigrationURL:         "file://db/migration",
		RedisAddress:         "localhost:56379",
		SMTPServerAddress:    "localhost:1025",
		EmailSenderName:      "SimpleBank",
		EmailSenderAddress:   "noreply@simplebank.dev",
		HTTPServerAddress:    "127.0.0.1:58080",
		GRPCServerAddress:    "127.0.0.1:59090",
		TokenSymmetricKey:    "too-short",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
	}

	require.Error(t, config.Validate())
}

func TestConfigValidateRejectsDevelopmentTokenKeyInProduction(t *testing.T) {
	config := Config{
		Environment:          "production",
		DBDriver:             "postgres",
		DBSource:             "postgresql://user:password@localhost:55432/simple_bank?sslmode=disable",
		MigrationURL:         "file://db/migration",
		RedisAddress:         "localhost:56379",
		SMTPServerAddress:    "localhost:1025",
		EmailSenderName:      "SimpleBank",
		EmailSenderAddress:   "noreply@simplebank.dev",
		HTTPServerAddress:    "127.0.0.1:58080",
		GRPCServerAddress:    "127.0.0.1:59090",
		TokenSymmetricKey:    developmentTokenSymmetricKey,
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
	}

	require.Error(t, config.Validate())
}

func TestConfigValidateRejectsPartialSMTPAuth(t *testing.T) {
	config := Config{
		Environment:          "test",
		DBDriver:             "postgres",
		DBSource:             "postgresql://user:password@localhost:55432/simple_bank?sslmode=disable",
		MigrationURL:         "file://db/migration",
		RedisAddress:         "localhost:56379",
		SMTPServerAddress:    "localhost:1025",
		EmailSenderName:      "SimpleBank",
		EmailSenderAddress:   "noreply@simplebank.dev",
		SMTPUsername:         "username",
		HTTPServerAddress:    "127.0.0.1:58080",
		GRPCServerAddress:    "127.0.0.1:59090",
		TokenSymmetricKey:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
	}

	require.Error(t, config.Validate())
}
