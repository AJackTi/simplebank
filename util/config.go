package util

import (
	"errors"
	"fmt"
	stdmail "net/mail"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config store all configuration of the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	MigrationURL         string        `mapstructure:"MIGRATION_URL"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	SMTPServerAddress    string        `mapstructure:"SMTP_SERVER_ADDRESS"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	SMTPUsername         string        `mapstructure:"SMTP_USERNAME"`
	SMTPPassword         string        `mapstructure:"SMTP_PASSWORD"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

var ErrInvalidConfig = errors.New("invalid configuration")

const developmentTokenSymmetricKey = "local-dev-secret-change-me-00000"

// Validate checks the values required to start the application safely.
func (config Config) Validate() error {
	for name, value := range map[string]string{
		"ENVIRONMENT":          config.Environment,
		"DB_DRIVER":            config.DBDriver,
		"DB_SOURCE":            config.DBSource,
		"MIGRATION_URL":        config.MigrationURL,
		"REDIS_ADDRESS":        config.RedisAddress,
		"SMTP_SERVER_ADDRESS":  config.SMTPServerAddress,
		"EMAIL_SENDER_NAME":    config.EmailSenderName,
		"EMAIL_SENDER_ADDRESS": config.EmailSenderAddress,
		"HTTP_SERVER_ADDRESS":  config.HTTPServerAddress,
		"GRPC_SERVER_ADDRESS":  config.GRPCServerAddress,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", ErrInvalidConfig, name)
		}
	}

	if len(config.TokenSymmetricKey) != 32 {
		return fmt.Errorf("%w: TOKEN_SYMMETRIC_KEY must be exactly 32 characters", ErrInvalidConfig)
	}
	if _, err := stdmail.ParseAddress(config.EmailSenderAddress); err != nil {
		return fmt.Errorf("%w: EMAIL_SENDER_ADDRESS must be a valid email address: %v", ErrInvalidConfig, err)
	}
	hasSMTPUsername := strings.TrimSpace(config.SMTPUsername) != ""
	hasSMTPPassword := strings.TrimSpace(config.SMTPPassword) != ""
	if hasSMTPUsername != hasSMTPPassword {
		return fmt.Errorf("%w: SMTP_USERNAME and SMTP_PASSWORD must be set together", ErrInvalidConfig)
	}
	if !isDevelopmentEnvironment(config.Environment) && config.TokenSymmetricKey == developmentTokenSymmetricKey {
		return fmt.Errorf("%w: the development TOKEN_SYMMETRIC_KEY cannot be used in %s", ErrInvalidConfig, config.Environment)
	}
	if config.AccessTokenDuration <= 0 || config.RefreshTokenDuration <= 0 {
		return fmt.Errorf("%w: token durations must be positive", ErrInvalidConfig)
	}
	if config.RefreshTokenDuration <= config.AccessTokenDuration {
		return fmt.Errorf("%w: refresh token duration must exceed access token duration", ErrInvalidConfig)
	}

	return nil
}

func isDevelopmentEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "dev", "test":
		return true
	default:
		return false
	}
}

// LoadConfig reads configuration from file or environment variable.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("app")
	v.SetConfigType("env")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for _, key := range []string{
		"ENVIRONMENT",
		"DB_DRIVER",
		"DB_SOURCE",
		"MIGRATION_URL",
		"REDIS_ADDRESS",
		"SMTP_SERVER_ADDRESS",
		"EMAIL_SENDER_NAME",
		"EMAIL_SENDER_ADDRESS",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
		"HTTP_SERVER_ADDRESS",
		"GRPC_SERVER_ADDRESS",
		"TOKEN_SYMMETRIC_KEY",
		"ACCESS_TOKEN_DURATION",
		"REFRESH_TOKEN_DURATION",
	} {
		if err := v.BindEnv(key); err != nil {
			return config, err
		}
	}

	if err = v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return config, err
		}
	}

	err = v.Unmarshal(&config)
	if err != nil {
		return config, err
	}
	err = config.Validate()
	return
}
