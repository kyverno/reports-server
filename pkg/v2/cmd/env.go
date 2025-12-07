package cmd

import (
	"os"
	"strconv"
)

// Environment variable names for database configuration
const (
	EnvDBHost     = "DB_HOST"
	EnvDBPort     = "DB_PORT"
	EnvDBUser     = "DB_USER"
	EnvDBPassword = "DB_PASSWORD"
	EnvDBDatabase = "DB_DATABASE"
)

// loadPostgresFromEnv loads PostgreSQL configuration from environment variables
// Command-line flags take precedence over environment variables
func (o *Options) loadPostgresFromEnv() {
	if o.DBHost == "" {
		o.DBHost = os.Getenv(EnvDBHost)
	}

	if o.DBName == "" {
		o.DBName = os.Getenv(EnvDBDatabase)
	}

	if o.DBUser == "" {
		o.DBUser = os.Getenv(EnvDBUser)
	}

	if o.DBPassword == "" {
		o.DBPassword = os.Getenv(EnvDBPassword)
	}

	// Load port if not set via flag
	if o.DBPort == DefaultDBPort {
		if portStr := os.Getenv(EnvDBPort); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				o.DBPort = port
			}
		}
	}
}
