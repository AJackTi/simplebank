package token

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingAuthorizationHeader       = errors.New("authorization header is not provided")
	ErrMultipleAuthorizationHeaders     = errors.New("multiple authorization headers are not allowed")
	ErrInvalidAuthorizationHeaderFormat = errors.New("invalid authorization header format")
	ErrUnsupportedAuthorizationType     = errors.New("unsupported authorization type")
)

const bearerAuthorizationType = "bearer"

func ExtractBearerToken(headers []string) (string, error) {
	switch len(headers) {
	case 0:
		return "", ErrMissingAuthorizationHeader
	case 1:
	default:
		return "", ErrMultipleAuthorizationHeaders
	}

	fields := strings.Fields(headers[0])
	if len(fields) != 2 {
		return "", ErrInvalidAuthorizationHeaderFormat
	}

	if strings.ToLower(fields[0]) != bearerAuthorizationType {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedAuthorizationType, fields[0])
	}

	return fields[1], nil
}
