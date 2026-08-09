package token

import "time"

// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken creates a new token for a specific username, type, and duration.
	CreateToken(username string, tokenType TokenType, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid and has the expected type.
	VerifyToken(token string, expectedType TokenType) (*Payload, error)
}
