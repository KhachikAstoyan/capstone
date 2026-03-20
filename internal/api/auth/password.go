package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	bcryptCost        = 12
)

var (
	ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", minPasswordLength)
	ErrPasswordTooLong  = fmt.Errorf("password must be at most %d characters", maxPasswordLength)
	ErrInvalidPassword  = errors.New("invalid password")
)

func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

func HashAndValidatePassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func ComparePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}
	return nil
}
