package models

import "errors"

var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrForbidden indicates the user does not have permission to access the resource.
	ErrForbidden = errors.New("access forbidden")

	// ErrUnauthorized indicates the request requires authentication.
	ErrUnauthorized = errors.New("authentication required")

	// ErrValidation indicates input validation failed.
	ErrValidation = errors.New("validation failed")

	// ErrDuplicateEmail indicates the email is already registered.
	ErrDuplicateEmail = errors.New("email already registered")

	// ErrInvalidCredentials indicates invalid email or password.
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// ValidationError carries field-specific error messages for form rendering.
type ValidationError struct {
	Fields map[string]string // field name → error message
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}
