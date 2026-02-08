package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// Error codes for consistent API responses
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeValidationFailed    = "VALIDATION_FAILED"
	CodeInternalError       = "INTERNAL_ERROR"
	CodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	CodeTOTPRequired        = "TOTP_REQUIRED"
	CodeInvalidTOTP         = "INVALID_TOTP"
	CodeTOTPAlreadyEnabled  = "TOTP_ALREADY_ENABLED"
	CodeTOTPNotEnabled      = "TOTP_NOT_ENABLED"
	CodeEmailTaken          = "EMAIL_TAKEN"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeSessionExpired      = "SESSION_EXPIRED"
	CodeFileNotFound        = "FILE_NOT_FOUND"
	CodeFileTooLarge        = "FILE_TOO_LARGE"
	CodeInvalidFileType     = "INVALID_FILE_TYPE"
	CodeStorageError        = "STORAGE_ERROR"
	CodeEmailSyncError      = "EMAIL_SYNC_ERROR"
	CodeEmailConnectionFail = "EMAIL_CONNECTION_FAILED"
)

// AppError is a structured application error
type AppError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	HTTPStatus int               `json:"-"`
	Internal   error             `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the internal error
func (e *AppError) Unwrap() error {
	return e.Internal
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key, value string) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithInternal sets the internal error (for logging)
func (e *AppError) WithInternal(err error) *AppError {
	e.Internal = err
	return e
}

// Error constructors

// BadRequest creates a 400 Bad Request error
func BadRequest(message string) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NotFound creates a 404 Not Found error
func NotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// Conflict creates a 409 Conflict error
func Conflict(message string) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

// ValidationFailed creates a 400 validation error
func ValidationFailed(field, message string) *AppError {
	return &AppError{
		Code:       CodeValidationFailed,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    map[string]string{"field": field},
	}
}

// InternalError creates a 500 Internal Server Error
func InternalError(message string) *AppError {
	return &AppError{
		Code:       CodeInternalError,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// RateLimitExceeded creates a 429 Too Many Requests error
func RateLimitExceeded() *AppError {
	return &AppError{
		Code:       CodeRateLimitExceeded,
		Message:    "Rate limit exceeded. Please try again later.",
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(service string) *AppError {
	return &AppError{
		Code:       CodeServiceUnavailable,
		Message:    fmt.Sprintf("%s is temporarily unavailable", service),
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// Domain-specific errors

// TOTPRequired creates an error indicating 2FA is required
func TOTPRequired() *AppError {
	return &AppError{
		Code:       CodeTOTPRequired,
		Message:    "Two-factor authentication code required",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// InvalidTOTP creates an error for invalid TOTP code
func InvalidTOTP() *AppError {
	return &AppError{
		Code:       CodeInvalidTOTP,
		Message:    "Invalid two-factor authentication code",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// TOTPAlreadyEnabled creates an error when 2FA is already enabled
func TOTPAlreadyEnabled() *AppError {
	return &AppError{
		Code:       CodeTOTPAlreadyEnabled,
		Message:    "Two-factor authentication is already enabled",
		HTTPStatus: http.StatusConflict,
	}
}

// TOTPNotEnabled creates an error when 2FA is not enabled
func TOTPNotEnabled() *AppError {
	return &AppError{
		Code:       CodeTOTPNotEnabled,
		Message:    "Two-factor authentication is not enabled",
		HTTPStatus: http.StatusBadRequest,
	}
}

// EmailTaken creates an error when email is already registered
func EmailTaken() *AppError {
	return &AppError{
		Code:       CodeEmailTaken,
		Message:    "Email address is already registered",
		HTTPStatus: http.StatusConflict,
	}
}

// InvalidCredentials creates an error for invalid login credentials
func InvalidCredentials() *AppError {
	return &AppError{
		Code:       CodeInvalidCredentials,
		Message:    "Invalid email or password",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// SessionExpired creates an error for expired session
func SessionExpired() *AppError {
	return &AppError{
		Code:       CodeSessionExpired,
		Message:    "Session has expired. Please log in again.",
		HTTPStatus: http.StatusUnauthorized,
	}
}

// FileNotFound creates an error for missing files
func FileNotFound() *AppError {
	return &AppError{
		Code:       CodeFileNotFound,
		Message:    "File not found",
		HTTPStatus: http.StatusNotFound,
	}
}

// FileTooLarge creates an error for oversized files
func FileTooLarge(maxSize string) *AppError {
	return &AppError{
		Code:       CodeFileTooLarge,
		Message:    fmt.Sprintf("File size exceeds maximum allowed size of %s", maxSize),
		HTTPStatus: http.StatusRequestEntityTooLarge,
	}
}

// InvalidFileType creates an error for unsupported file types
func InvalidFileType(fileType string) *AppError {
	return &AppError{
		Code:       CodeInvalidFileType,
		Message:    fmt.Sprintf("File type '%s' is not supported", fileType),
		HTTPStatus: http.StatusBadRequest,
	}
}

// StorageError creates an error for storage operations
func StorageError(operation string) *AppError {
	return &AppError{
		Code:       CodeStorageError,
		Message:    fmt.Sprintf("Storage %s failed", operation),
		HTTPStatus: http.StatusInternalServerError,
	}
}

// EmailSyncError creates an error for email sync failures
func EmailSyncError(message string) *AppError {
	return &AppError{
		Code:       CodeEmailSyncError,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// EmailConnectionFailed creates an error for email connection failures
func EmailConnectionFailed() *AppError {
	return &AppError{
		Code:       CodeEmailConnectionFail,
		Message:    "Failed to connect to email server",
		HTTPStatus: http.StatusBadGateway,
	}
}

// Response helpers

// ErrorResponse is the standard error response format
type ErrorResponse struct {
	Success bool              `json:"success"`
	Error   ErrorDetail       `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// ToResponse converts an AppError to ErrorResponse
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    e.Code,
			Message: e.Message,
			Details: e.Details,
		},
	}
}

// SendError sends an error response
func SendError(c *fiber.Ctx, err error, log zerolog.Logger) error {
	var appErr *AppError

	if errors.As(err, &appErr) {
		// Log internal error if present
		if appErr.Internal != nil {
			log.Error().
				Err(appErr.Internal).
				Str("code", appErr.Code).
				Str("message", appErr.Message).
				Msg("Application error")
		}
		return c.Status(appErr.HTTPStatus).JSON(appErr.ToResponse())
	}

	// Handle fiber errors
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		appErr = &AppError{
			Code:       CodeInternalError,
			Message:    fiberErr.Message,
			HTTPStatus: fiberErr.Code,
		}
		return c.Status(appErr.HTTPStatus).JSON(appErr.ToResponse())
	}

	// Unknown error - log and return generic message
	log.Error().Err(err).Msg("Unexpected error")
	appErr = InternalError("An unexpected error occurred")
	return c.Status(appErr.HTTPStatus).JSON(appErr.ToResponse())
}

// ErrorHandler is a Fiber error handler for global error handling
func ErrorHandler(log zerolog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		return SendError(c, err, log)
	}
}

// Helper to check error types

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNotFound
	}
	return false
}

// IsUnauthorized checks if error is an unauthorized error
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeUnauthorized || appErr.Code == CodeInvalidCredentials
	}
	return false
}

// IsTOTPRequired checks if error indicates TOTP is required
func IsTOTPRequired(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeTOTPRequired
	}
	return false
}

// WrapError wraps a standard error as an internal error
func WrapError(err error, message string) *AppError {
	return InternalError(message).WithInternal(err)
}
