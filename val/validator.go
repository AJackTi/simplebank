package val

import (
	"fmt"
	"net/mail"
	"regexp"

	"github.com/AJackTi/simplebank/util"
)

var (
	isValidUsername = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
	isValidFullName = regexp.MustCompile(`^[a-zA-Z\s]+$`).MatchString
)

func ValidateString(value string, minLength int, maxLength int) error {
	n := len(value)
	if n < minLength || n > maxLength {
		return fmt.Errorf("must contain from %d-%d characters", minLength, maxLength)
	}
	return nil
}

func ValidateUsername(value string) error {
	if err := ValidateString(value, 3, 10); err != nil {
		return err
	}
	if !isValidUsername(value) {
		return fmt.Errorf("must contain only lowercase letters, digits, or underscore")
	}
	return nil
}

func ValidateFullName(value string) error {
	if err := ValidateString(value, 3, 10); err != nil {
		return err
	}
	if !isValidFullName(value) {
		return fmt.Errorf("must contain only letters or spaces")
	}
	return nil
}

func ValidatePassword(value string) error {
	return ValidateString(value, 6, 100)
}

func ValidateEmail(value string) error {
	if err := ValidateString(value, 3, 200); err != nil {
		return err
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return fmt.Errorf("is not a valid email address")
	}
	return nil
}

func ValidateAccountID(value int64) error {
	if value < 1 {
		return fmt.Errorf("must be at least 1")
	}
	return nil
}

func ValidateTransferAmount(value int64) error {
	if value <= 0 {
		return fmt.Errorf("must be greater than 0")
	}
	return nil
}

func ValidateCurrency(value string) error {
	if !util.IsSupportedCurrency(value) {
		return fmt.Errorf("unsupported currency")
	}
	return nil
}

func ValidatePageID(value int32) error {
	if value < 1 {
		return fmt.Errorf("must be at least 1")
	}
	return nil
}

func ValidatePageSize(value int32) error {
	if value < 5 || value > 10 {
		return fmt.Errorf("must be between 5 and 10")
	}
	return nil
}
