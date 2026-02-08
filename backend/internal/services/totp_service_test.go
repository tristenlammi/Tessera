package services

import (
	"testing"
)

func TestTOTPService_GenerateSecret(t *testing.T) {
	service := NewTOTPService()

	t.Run("generates valid secret", func(t *testing.T) {
		secret, err := service.GenerateSecret()
		if err != nil {
			t.Fatalf("GenerateSecret() error = %v", err)
		}
		if len(secret) == 0 {
			t.Error("GenerateSecret() returned empty secret")
		}
		// Base32 encoded 20-byte secret should be 32 characters
		if len(secret) != 32 {
			t.Errorf("GenerateSecret() secret length = %d, want 32", len(secret))
		}
	})

	t.Run("generates unique secrets", func(t *testing.T) {
		secrets := make(map[string]bool)
		for i := 0; i < 100; i++ {
			secret, err := service.GenerateSecret()
			if err != nil {
				t.Fatalf("GenerateSecret() error = %v", err)
			}
			if secrets[secret] {
				t.Error("GenerateSecret() generated duplicate secret")
			}
			secrets[secret] = true
		}
	})
}

func TestTOTPService_ValidateCode(t *testing.T) {
	service := NewTOTPService()

	t.Run("rejects invalid code format", func(t *testing.T) {
		secret, _ := service.GenerateSecret()

		// Statistically, "000000" is very unlikely to be valid
		// We're mainly testing the code validation logic
		if service.ValidateCode(secret, "") {
			t.Error("ValidateCode() accepted empty code")
		}
	})

	t.Run("rejects wrong length code", func(t *testing.T) {
		secret, _ := service.GenerateSecret()

		if service.ValidateCode(secret, "12345") {
			t.Error("ValidateCode() accepted 5-digit code")
		}
		if service.ValidateCode(secret, "1234567") {
			t.Error("ValidateCode() accepted 7-digit code")
		}
	})
}

func TestTOTPService_GenerateOTPAuthURL(t *testing.T) {
	service := NewTOTPService()

	t.Run("generates valid otpauth URL", func(t *testing.T) {
		secret, _ := service.GenerateSecret()
		url := service.GenerateOTPAuthURL(secret, "test@example.com")

		expected := "otpauth://totp/Tessera:test@example.com?secret=" + secret + "&issuer=Tessera&algorithm=SHA1&digits=6&period=30"
		if url != expected {
			t.Errorf("GenerateOTPAuthURL() = %s, want %s", url, expected)
		}
	})
}

func TestTOTPService_FormatSecretForDisplay(t *testing.T) {
	service := NewTOTPService()

	t.Run("formats secret in groups of 4", func(t *testing.T) {
		secret := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" // 32 characters
		formatted := service.FormatSecretForDisplay(secret)

		expected := "ABCD EFGH IJKL MNOP QRST UVWX YZ23 4567"
		if formatted != expected {
			t.Errorf("FormatSecretForDisplay() = %s, want %s", formatted, expected)
		}
	})

	t.Run("handles short secrets", func(t *testing.T) {
		secret := "ABCDEF"
		formatted := service.FormatSecretForDisplay(secret)

		expected := "ABCD EF"
		if formatted != expected {
			t.Errorf("FormatSecretForDisplay() = %s, want %s", formatted, expected)
		}
	})
}

func TestTOTPService_GenerateBackupCodes(t *testing.T) {
	service := NewTOTPService()

	t.Run("generates requested number of backup codes", func(t *testing.T) {
		codes, hashedCodes, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		if len(codes) != 10 {
			t.Errorf("GenerateBackupCodes() codes count = %d, want 10", len(codes))
		}

		if len(hashedCodes) != 10 {
			t.Errorf("GenerateBackupCodes() hashed codes count = %d, want 10", len(hashedCodes))
		}
	})

	t.Run("generates unique codes", func(t *testing.T) {
		codes, _, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		seen := make(map[string]bool)
		for _, code := range codes {
			if seen[code] {
				t.Error("GenerateBackupCodes() generated duplicate code")
			}
			seen[code] = true
		}
	})

	t.Run("backup codes contain hyphen", func(t *testing.T) {
		codes, _, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		for _, code := range codes {
			// Format should contain a hyphen
			found := false
			for _, c := range code {
				if c == '-' {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GenerateBackupCodes() code missing hyphen: %s", code)
			}
		}
	})
}

func TestTOTPService_ValidateBackupCode(t *testing.T) {
	service := NewTOTPService()

	t.Run("validates correct backup code", func(t *testing.T) {
		codes, hashedCodes, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		// Test first code - returns index
		idx := service.ValidateBackupCode(codes[0], hashedCodes)
		if idx != 0 {
			t.Errorf("ValidateBackupCode() index = %d, want 0", idx)
		}

		// Test last code
		idx = service.ValidateBackupCode(codes[9], hashedCodes)
		if idx != 9 {
			t.Errorf("ValidateBackupCode() index = %d, want 9", idx)
		}
	})

	t.Run("rejects invalid backup code", func(t *testing.T) {
		_, hashedCodes, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		idx := service.ValidateBackupCode("INVALID1-CODE1234", hashedCodes)
		if idx != -1 {
			t.Error("ValidateBackupCode() accepted invalid code")
		}
	})

	t.Run("rejects empty backup code", func(t *testing.T) {
		_, hashedCodes, err := service.GenerateBackupCodes(10)
		if err != nil {
			t.Fatalf("GenerateBackupCodes() error = %v", err)
		}

		idx := service.ValidateBackupCode("", hashedCodes)
		if idx != -1 {
			t.Error("ValidateBackupCode() accepted empty code")
		}
	})
}

// Benchmark tests
func BenchmarkValidateCode(b *testing.B) {
	service := NewTOTPService()
	secret, _ := service.GenerateSecret()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ValidateCode(secret, "123456")
	}
}
