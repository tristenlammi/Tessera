package middleware

import (
	"testing"
)

func TestValidator_ValidateStrongPassword(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"valid strong password", "Password123!", true},
		{"valid with symbols", "Test@1234", true},
		{"no uppercase", "password123!", false},
		{"no lowercase", "PASSWORD123!", false},
		{"no number", "PasswordTest!", false},
		{"no special char", "Password1234", false},
		{"too short", "Pass1!", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.password, "strong_password")
			got := err == nil
			if got != tt.want {
				t.Errorf("strong_password validation for %q = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateSafeString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"normal text", "Hello World", true},
		{"with numbers", "Test123", true},
		{"with punctuation", "Hello, World!", true},
		{"script tag", "<script>alert('xss')</script>", false},
		{"javascript protocol", "javascript:alert(1)", false},
		{"onclick handler", "onclick=alert(1)", false},
		{"sql comment", "test--comment", false},
		{"sql injection", "'; DROP TABLE users;--", false},
		{"union select", "test UNION SELECT * FROM users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.input, "safe_string")
			got := err == nil
			if got != tt.want {
				t.Errorf("safe_string validation for %q = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateUUID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{"valid uuid v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uuid", "123e4567-e89b-12d3-a456-426614174000", true},
		{"empty allowed", "", true},
		{"invalid format", "not-a-uuid", false},
		{"missing hyphens still valid", "550e8400e29b41d4a716446655440000", true}, // uuid.Parse accepts this
		{"too short", "550e8400-e29b-41d4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.uuid, "valid_uuid")
			got := err == nil
			if got != tt.want {
				t.Errorf("valid_uuid validation for %q = %v, want %v", tt.uuid, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateEmail(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"empty allowed", "", true},
		{"no at symbol", "notanemail", false},
		{"no domain", "user@", false},
		{"no user", "@example.com", false},
		{"spaces", "user @example.com", false},
		{"double at", "user@@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.email, "valid_email")
			got := err == nil
			if got != tt.want {
				t.Errorf("valid_email validation for %q = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateNoHTML(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"normal text", "Hello World", true},
		{"with numbers", "Test 123", true},
		{"angle brackets without tag", "a < b > c", false}, // This is tricky, but we're being safe
		{"html tag", "<div>content</div>", false},
		{"script tag", "<script>alert(1)</script>", false},
		{"self-closing tag", "<br/>", false},
		{"img tag", "<img src=x>", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.input, "no_html")
			got := err == nil
			if got != tt.want {
				t.Errorf("no_html validation for %q = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateSafeFilename(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"valid filename", "document.pdf", true},
		{"with underscores", "my_file.txt", true},
		{"with hyphens", "my-file.txt", true},
		{"with numbers", "file123.doc", true},
		{"empty allowed", "", true},
		{"path traversal", "../etc/passwd", false},
		{"path traversal windows", "..\\system32\\config", false},
		{"absolute path", "/etc/passwd", false},
		{"null byte", "file\x00.txt", false},
		{"spaces", "my file.txt", false},
		{"special chars", "file@#$.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.filename, "safe_filename")
			got := err == nil
			if got != tt.want {
				t.Errorf("safe_filename validation for %q = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateAlphaSpace(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"letters only", "JohnDoe", true},
		{"with spaces", "John Doe", true},
		{"unicode letters", "José María", true},
		{"with numbers", "John123", false},
		{"with symbols", "John@Doe", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.input, "alpha_space")
			got := err == nil
			if got != tt.want {
				t.Errorf("alpha_space validation for %q = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateURL(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"https url", "https://example.com", true},
		{"http url", "http://example.com", true},
		{"with path", "https://example.com/path", true},
		{"with subdomain", "https://sub.example.com", true},
		{"without protocol", "example.com", true},
		{"empty allowed", "", true},
		{"invalid tld", "https://example", false},
		{"just protocol", "https://", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVar(tt.url, "valid_url")
			got := err == nil
			if got != tt.want {
				t.Errorf("valid_url validation for %q = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateStruct(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Email    string `json:"email" validate:"required,valid_email"`
		Password string `json:"password" validate:"required,strong_password"`
		Name     string `json:"name" validate:"required,min=2,max=50"`
	}

	t.Run("valid struct", func(t *testing.T) {
		s := TestStruct{
			Email:    "test@example.com",
			Password: "Password123!",
			Name:     "John Doe",
		}

		errs := v.Validate(&s)
		if errs != nil {
			t.Errorf("Validate() returned errors for valid struct: %v", errs)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		s := TestStruct{}

		errs := v.Validate(&s)
		if errs == nil {
			t.Error("Validate() should return errors for empty struct")
		}
		if len(errs.Errors) != 3 {
			t.Errorf("Validate() returned %d errors, want 3", len(errs.Errors))
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		s := TestStruct{
			Email:    "invalid-email",
			Password: "Password123!",
			Name:     "John",
		}

		errs := v.Validate(&s)
		if errs == nil {
			t.Error("Validate() should return error for invalid email")
		}
	})

	t.Run("weak password", func(t *testing.T) {
		s := TestStruct{
			Email:    "test@example.com",
			Password: "weak",
			Name:     "John",
		}

		errs := v.Validate(&s)
		if errs == nil {
			t.Error("Validate() should return error for weak password")
		}
	})

	t.Run("name too short", func(t *testing.T) {
		s := TestStruct{
			Email:    "test@example.com",
			Password: "Password123!",
			Name:     "J",
		}

		errs := v.Validate(&s)
		if errs == nil {
			t.Error("Validate() should return error for short name")
		}
	})
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal text", "Hello World", "Hello World"},
		{"with leading/trailing spaces", "  Hello  ", "Hello"},
		{"with null bytes", "Hello\x00World", "HelloWorld"},
		{"with control chars", "Hello\x01\x02World", "HelloWorld"},
		{"with newlines", "Hello\nWorld", "Hello\nWorld"},
		{"with tabs", "Hello\tWorld", "Hello\tWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already lowercase", "test@example.com", "test@example.com"},
		{"mixed case", "Test@Example.COM", "test@example.com"},
		{"with spaces", "  test@example.com  ", "test@example.com"},
		{"uppercase", "TEST@EXAMPLE.COM", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeEmail(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
