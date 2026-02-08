package middleware

import (
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Validator is a custom validator for request validation
type Validator struct {
	validate *validator.Validate
}

// ValidationError represents a validation error for a single field
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewValidator creates a new validator instance with custom validations
func NewValidator() *Validator {
	v := validator.New()

	// Use JSON tag names in error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validators
	v.RegisterValidation("strong_password", validateStrongPassword)
	v.RegisterValidation("safe_string", validateSafeString)
	v.RegisterValidation("valid_uuid", validateUUID)
	v.RegisterValidation("valid_email", validateEmail)
	v.RegisterValidation("no_html", validateNoHTML)
	v.RegisterValidation("alpha_space", validateAlphaSpace)
	v.RegisterValidation("safe_filename", validateSafeFilename)
	v.RegisterValidation("valid_url", validateURL)

	return &Validator{validate: v}
}

// Validate validates a struct and returns formatted errors
func (v *Validator) Validate(i interface{}) *ValidationErrors {
	err := v.validate.Struct(i)
	if err == nil {
		return nil
	}

	var errors []ValidationError

	for _, err := range err.(validator.ValidationErrors) {
		errors = append(errors, ValidationError{
			Field:   err.Field(),
			Message: formatValidationError(err),
		})
	}

	return &ValidationErrors{Errors: errors}
}

// ValidateVar validates a single variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// formatValidationError returns a human-readable error message
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "email", "valid_email":
		return fmt.Sprintf("%s must be a valid email address", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", err.Field(), err.Param())
	case "strong_password":
		return "Password must be at least 8 characters with uppercase, lowercase, number, and special character"
	case "safe_string":
		return fmt.Sprintf("%s contains invalid characters", err.Field())
	case "valid_uuid":
		return fmt.Sprintf("%s must be a valid UUID", err.Field())
	case "no_html":
		return fmt.Sprintf("%s cannot contain HTML tags", err.Field())
	case "alpha_space":
		return fmt.Sprintf("%s can only contain letters and spaces", err.Field())
	case "safe_filename":
		return fmt.Sprintf("%s contains invalid filename characters", err.Field())
	case "valid_url":
		return fmt.Sprintf("%s must be a valid URL", err.Field())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", err.Field(), err.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", err.Field(), err.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", err.Field(), err.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s", err.Field(), err.Tag())
	}
}

// validateStrongPassword validates password strength
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// validateSafeString validates that string doesn't contain dangerous characters
func validateSafeString(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	// Block common SQL injection and XSS patterns
	dangerous := []string{"<script", "javascript:", "onerror=", "onclick=", "--", "/*", "*/", ";--", "';", "\"'", "or 1=1", "union select"}
	lower := strings.ToLower(str)
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return false
		}
	}
	return true
}

// validateUUID validates UUID format
func validateUUID(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" {
		return true // Let required handle empty
	}
	_, err := uuid.Parse(str)
	return err == nil
}

// validateEmail validates email format more strictly
func validateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	if email == "" {
		return true // Let required handle empty
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	// Additional checks
	if len(email) > 254 {
		return false
	}
	// Basic pattern check
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// validateNoHTML validates that string doesn't contain HTML tags
func validateNoHTML(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	htmlPattern := `<[^>]*>`
	matched, _ := regexp.MatchString(htmlPattern, str)
	return !matched
}

// validateAlphaSpace validates that string only contains letters and spaces
func validateAlphaSpace(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	for _, c := range str {
		if !unicode.IsLetter(c) && !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

// validateSafeFilename validates that string is a safe filename
func validateSafeFilename(fl validator.FieldLevel) bool {
	filename := fl.Field().String()
	if filename == "" {
		return true
	}
	// Block path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}
	// Block null bytes
	if strings.Contains(filename, "\x00") {
		return false
	}
	// Only allow safe characters
	safePattern := `^[a-zA-Z0-9._-]+$`
	matched, _ := regexp.MatchString(safePattern, filename)
	return matched
}

// validateURL validates URL format
func validateURL(fl validator.FieldLevel) bool {
	url := fl.Field().String()
	if url == "" {
		return true
	}
	urlPattern := `^(https?://)?[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}(/.*)?$`
	matched, _ := regexp.MatchString(urlPattern, url)
	return matched
}

// ValidationMiddleware creates a middleware that validates request bodies
func ValidationMiddleware(v *Validator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("validator", v)
		return c.Next()
	}
}

// GetValidator retrieves the validator from context
func GetValidator(c *fiber.Ctx) *Validator {
	if v, ok := c.Locals("validator").(*Validator); ok {
		return v
	}
	return NewValidator()
}

// ValidateBody parses and validates request body
func ValidateBody[T any](c *fiber.Ctx) (*T, error) {
	var req T
	if err := c.BodyParser(&req); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	v := GetValidator(c)
	if errs := v.Validate(&req); errs != nil {
		return nil, &fiber.Error{
			Code:    fiber.StatusBadRequest,
			Message: errs.Errors[0].Message,
		}
	}

	return &req, nil
}

// SanitizeString removes potentially dangerous content from a string
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Remove control characters except newline and tab
	var result strings.Builder
	for _, c := range s {
		if c == '\n' || c == '\t' || c >= 32 {
			result.WriteRune(c)
		}
	}
	// Trim whitespace
	return strings.TrimSpace(result.String())
}

// SanitizeEmail normalizes and validates an email address
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	return email
}
