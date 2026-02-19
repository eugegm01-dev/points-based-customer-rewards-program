// Package config provides application configuration loading from
// command-line flags and environment variables.
//
// Configuration sources are resolved in the following order of precedence
// (highest first): flags, environment variables, defaults.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration.
// All fields are set from flags or environment variables as specified in the Gophermart API spec.
type Config struct {
	// RunAddress is the host:port the HTTP server listens on (e.g. ":8080").
	// Set via -a flag or RUN_ADDRESS env.
	RunAddress string

	// DatabaseURI is the PostgreSQL connection string (DSN).
	// Set via -d flag or DATABASE_URI env.
	DatabaseURI string

	// AccrualAddress is the base URL of the accrual system API.
	// Set via -r flag or ACCRUAL_SYSTEM_ADDRESS env.
	AccrualAddress string

	// AuthSecret is the secret key for signing JWT (e.g. auth cookie).
	// Set via -s flag or AUTH_SECRET env. Required for auth.
	AuthSecret string
}

const (
	envRunAddress     = "RUN_ADDRESS"
	envDatabaseURI    = "DATABASE_URI"
	envAccrualAddress = "ACCRUAL_SYSTEM_ADDRESS"
	envAuthSecret     = "AUTH_SECRET"

	defaultRunAddress = ":8080"
)

// Load parses flags and environment variables, then builds and validates Config.
// Flags override environment variables. Returns an error if required fields are missing.
func Load() (*Config, error) {
	runAddr := flag.String("a", "", "Server address and port (e.g. :8080). Overrides RUN_ADDRESS.")
	dbURI := flag.String("d", "", "PostgreSQL connection URI. Overrides DATABASE_URI.")
	accrualAddr := flag.String("r", "", "Accrual system base URL. Overrides ACCRUAL_SYSTEM_ADDRESS.")
	authSecret := flag.String("s", "", "Secret for JWT signing. Overrides AUTH_SECRET.")

	flag.Parse()

	cfg := &Config{
		RunAddress:     firstNonEmpty(*runAddr, os.Getenv(envRunAddress), defaultRunAddress),
		DatabaseURI:    firstNonEmpty(*dbURI, os.Getenv(envDatabaseURI)),
		AccrualAddress: firstNonEmpty(*accrualAddr, os.Getenv(envAccrualAddress)),
		AuthSecret:     firstNonEmpty(*authSecret, os.Getenv(envAuthSecret)),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required configuration fields are set and well-formed.
func (c *Config) validate() error {
	var missing []string
	if strings.TrimSpace(c.DatabaseURI) == "" {
		missing = append(missing, fmt.Sprintf("database URI (flag -d or %s)", envDatabaseURI))
	}
	if strings.TrimSpace(c.AccrualAddress) == "" {
		missing = append(missing, fmt.Sprintf("accrual address (flag -r or %s)", envAccrualAddress))
	}
	if strings.TrimSpace(c.AuthSecret) == "" {
		missing = append(missing, fmt.Sprintf("auth secret (flag -s or %s)", envAuthSecret))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrMissingConfig, strings.Join(missing, ", "))
	}

	c.AccrualAddress = strings.TrimSuffix(c.AccrualAddress, "/")
	return nil
}

// firstNonEmpty returns the first non-empty string from the given values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ErrMissingConfig indicates one or more required configuration values are missing.
var ErrMissingConfig = errors.New("missing required config")
