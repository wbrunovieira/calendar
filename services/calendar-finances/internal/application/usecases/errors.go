package usecases

import "errors"

var (
	ErrProfileAlreadyExists = errors.New("profile already exists for this calendar")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrInvalidInput         = errors.New("invalid input")
)
