package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

var (
	runAddrFlag     = flag.String("a", "", "Server address and port (e.g. :8080). Overrides RUN_ADDRESS.")
	dbURIFlag       = flag.String("d", "", "PostgreSQL connection URI. Overrides DATABASE_URI.")
	accrualAddrFlag = flag.String("r", "", "Accrual system base URL. Overrides ACCRUAL_SYSTEM_ADDRESS.")
	authSecretFlag  = flag.String("s", "", "Secret for JWT signing. Overrides AUTH_SECRET.")
)

func init() {
	// Only parse flags if not in test mode
	if !testing.Testing() {
		flag.Parse()
	}
}

// Config holds all application configuration.
type Config struct {
	RunAddress     string
	DatabaseURI    string
	AccrualAddress string
	AuthSecret     string
}

const (
	envRunAddress      = "RUN_ADDRESS"
	envDatabaseURI     = "DATABASE_URI"
	envAccrualAddress  = "ACCRUAL_SYSTEM_ADDRESS"
	envAuthSecret      = "AUTH_SECRET"
	defaultRunAddress  = ":8080"
	defaultAuthSecret  = "autotest-secret-key-change-in-production"
	defaultAccrualAddr = "http://localhost:8080"
)

// Load parses flags and environment variables, then builds and validates Config.
func Load() (*Config, error) {
	cfg := &Config{
		RunAddress:     firstNonEmpty(*runAddrFlag, os.Getenv(envRunAddress), defaultRunAddress),
		DatabaseURI:    firstNonEmpty(*dbURIFlag, os.Getenv(envDatabaseURI)),
		AccrualAddress: firstNonEmpty(*accrualAddrFlag, os.Getenv(envAccrualAddress), defaultAccrualAddr),
		AuthSecret:     firstNonEmpty(*authSecretFlag, os.Getenv(envAuthSecret), defaultAuthSecret),
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
