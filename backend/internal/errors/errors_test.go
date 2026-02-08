package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError(t *testing.T) {
	t.Run("Error() returns formatted message", func(t *testing.T) {
		err := &AppError{
			Code:    CodeBadRequest,
			Message: "Test error",
		}

		expected := "BAD_REQUEST: Test error"
		if err.Error() != expected {
			t.Errorf("Error() = %s, want %s", err.Error(), expected)
		}
	})

	t.Run("Error() includes internal error", func(t *testing.T) {
		internalErr := errors.New("internal failure")
		err := &AppError{
			Code:     CodeInternalError,
			Message:  "Something went wrong",
			Internal: internalErr,
		}

		result := err.Error()
		if result == "" {
			t.Error("Error() returned empty string")
		}
		if !errors.Is(err, internalErr) {
			t.Error("Unwrap() should return internal error")
		}
	})

	t.Run("WithDetail adds detail", func(t *testing.T) {
		err := BadRequest("Invalid input").WithDetail("field", "email")

		if err.Details == nil {
			t.Fatal("Details should not be nil")
		}
		if err.Details["field"] != "email" {
			t.Errorf("Details[field] = %s, want email", err.Details["field"])
		}
	})

	t.Run("WithInternal sets internal error", func(t *testing.T) {
		internalErr := errors.New("database error")
		err := InternalError("Operation failed").WithInternal(internalErr)

		if err.Internal != internalErr {
			t.Error("WithInternal() did not set internal error")
		}
	})
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name           string
		constructor    func() *AppError
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "BadRequest",
			constructor:    func() *AppError { return BadRequest("test") },
			expectedCode:   CodeBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized",
			constructor:    func() *AppError { return Unauthorized("test") },
			expectedCode:   CodeUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden",
			constructor:    func() *AppError { return Forbidden("test") },
			expectedCode:   CodeForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "NotFound",
			constructor:    func() *AppError { return NotFound("Resource") },
			expectedCode:   CodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Conflict",
			constructor:    func() *AppError { return Conflict("test") },
			expectedCode:   CodeConflict,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "InternalError",
			constructor:    func() *AppError { return InternalError("test") },
			expectedCode:   CodeInternalError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "RateLimitExceeded",
			constructor:    func() *AppError { return RateLimitExceeded() },
			expectedCode:   CodeRateLimitExceeded,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "ServiceUnavailable",
			constructor:    func() *AppError { return ServiceUnavailable("Database") },
			expectedCode:   CodeServiceUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "TOTPRequired",
			constructor:    func() *AppError { return TOTPRequired() },
			expectedCode:   CodeTOTPRequired,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "InvalidTOTP",
			constructor:    func() *AppError { return InvalidTOTP() },
			expectedCode:   CodeInvalidTOTP,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "TOTPAlreadyEnabled",
			constructor:    func() *AppError { return TOTPAlreadyEnabled() },
			expectedCode:   CodeTOTPAlreadyEnabled,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "TOTPNotEnabled",
			constructor:    func() *AppError { return TOTPNotEnabled() },
			expectedCode:   CodeTOTPNotEnabled,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "EmailTaken",
			constructor:    func() *AppError { return EmailTaken() },
			expectedCode:   CodeEmailTaken,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "InvalidCredentials",
			constructor:    func() *AppError { return InvalidCredentials() },
			expectedCode:   CodeInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "SessionExpired",
			constructor:    func() *AppError { return SessionExpired() },
			expectedCode:   CodeSessionExpired,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "FileNotFound",
			constructor:    func() *AppError { return FileNotFound() },
			expectedCode:   CodeFileNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "FileTooLarge",
			constructor:    func() *AppError { return FileTooLarge("100MB") },
			expectedCode:   CodeFileTooLarge,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "InvalidFileType",
			constructor:    func() *AppError { return InvalidFileType(".exe") },
			expectedCode:   CodeInvalidFileType,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "StorageError",
			constructor:    func() *AppError { return StorageError("upload") },
			expectedCode:   CodeStorageError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "EmailSyncError",
			constructor:    func() *AppError { return EmailSyncError("sync failed") },
			expectedCode:   CodeEmailSyncError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "EmailConnectionFailed",
			constructor:    func() *AppError { return EmailConnectionFailed() },
			expectedCode:   CodeEmailConnectionFail,
			expectedStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()

			if err.Code != tt.expectedCode {
				t.Errorf("%s().Code = %s, want %s", tt.name, err.Code, tt.expectedCode)
			}
			if err.HTTPStatus != tt.expectedStatus {
				t.Errorf("%s().HTTPStatus = %d, want %d", tt.name, err.HTTPStatus, tt.expectedStatus)
			}
			if err.Message == "" {
				t.Errorf("%s().Message is empty", tt.name)
			}
		})
	}
}

func TestValidationFailed(t *testing.T) {
	err := ValidationFailed("email", "Email is invalid")

	if err.Code != CodeValidationFailed {
		t.Errorf("Code = %s, want %s", err.Code, CodeValidationFailed)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusBadRequest)
	}
	if err.Details["field"] != "email" {
		t.Errorf("Details[field] = %s, want email", err.Details["field"])
	}
}

func TestToResponse(t *testing.T) {
	err := BadRequest("Invalid input").WithDetail("field", "email")
	response := err.ToResponse()

	if response.Success {
		t.Error("ToResponse().Success should be false")
	}
	if response.Error.Code != CodeBadRequest {
		t.Errorf("ToResponse().Error.Code = %s, want %s", response.Error.Code, CodeBadRequest)
	}
	if response.Error.Message != "Invalid input" {
		t.Errorf("ToResponse().Error.Message = %s, want Invalid input", response.Error.Message)
	}
	if response.Error.Details["field"] != "email" {
		t.Errorf("ToResponse().Error.Details[field] = %s, want email", response.Error.Details["field"])
	}
}

func TestErrorTypeChecks(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		notFoundErr := NotFound("User")
		otherErr := BadRequest("Invalid")

		if !IsNotFound(notFoundErr) {
			t.Error("IsNotFound() should return true for NotFound error")
		}
		if IsNotFound(otherErr) {
			t.Error("IsNotFound() should return false for other error")
		}
		if IsNotFound(errors.New("random error")) {
			t.Error("IsNotFound() should return false for non-AppError")
		}
	})

	t.Run("IsUnauthorized", func(t *testing.T) {
		unauthorizedErr := Unauthorized("Not authenticated")
		invalidCredErr := InvalidCredentials()
		otherErr := BadRequest("Invalid")

		if !IsUnauthorized(unauthorizedErr) {
			t.Error("IsUnauthorized() should return true for Unauthorized error")
		}
		if !IsUnauthorized(invalidCredErr) {
			t.Error("IsUnauthorized() should return true for InvalidCredentials error")
		}
		if IsUnauthorized(otherErr) {
			t.Error("IsUnauthorized() should return false for other error")
		}
	})

	t.Run("IsTOTPRequired", func(t *testing.T) {
		totpErr := TOTPRequired()
		otherErr := Unauthorized("Not authenticated")

		if !IsTOTPRequired(totpErr) {
			t.Error("IsTOTPRequired() should return true for TOTPRequired error")
		}
		if IsTOTPRequired(otherErr) {
			t.Error("IsTOTPRequired() should return false for other error")
		}
	})
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	wrappedErr := WrapError(originalErr, "Failed to fetch user")

	if wrappedErr.Code != CodeInternalError {
		t.Errorf("WrapError().Code = %s, want %s", wrappedErr.Code, CodeInternalError)
	}
	if wrappedErr.Message != "Failed to fetch user" {
		t.Errorf("WrapError().Message = %s, want 'Failed to fetch user'", wrappedErr.Message)
	}
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("WrapError() should wrap the original error")
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := InternalError("wrapper").WithInternal(originalErr)

	unwrapped := appErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Unwrap() should return internal error")
	}
}
