package config

import (
	"os"
	"testing"
)

func TestSessionSecretValidation(t *testing.T) {
	// Set valid database URL
	t.Setenv("DATABASE_URL", "postgres://localhost/eduplay_test")

	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty secret", "", true},
		{"short secret (10 chars)", "1234567890", true},
		{"short secret (31 chars)", "1234567890123456789012345678901", true},
		{"exact 32 chars", "12345678901234567890123456789012", false},
		{"long secret (> 32 chars)", "1234567890123456789012345678901234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SESSION_SECRET", tt.secret)
			t.Setenv("APP_ENV", "development")
			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseURLRequired(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	t.Setenv("SESSION_SECRET", "12345678901234567890123456789012")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is unset")
	}
}
