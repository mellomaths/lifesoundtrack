package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadLocalDotEnv loads a single .env from the process working directory (e.g. bot/ when
// developing). If the file is missing, it succeeds. If the file exists and cannot be read
// or parsed, it returns a non-wrapped error for invalid content (so callers can avoid
// logging sensitive values); missing-file and permission errors are wrapped.
func LoadLocalDotEnv() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	p := filepath.Join(wd, ".env")
	_, stErr := os.Stat(p)
	if stErr != nil {
		if errors.Is(stErr, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("access .env: %w", stErr)
	}
	// Existence and readability are required once the file is present; parse errors
	// are returned for operator diagnosis without echoing the file body.
	if err := godotenv.Load(p); err != nil {
		return fmt.Errorf("invalid .env file: %w", err)
	}
	return nil
}
