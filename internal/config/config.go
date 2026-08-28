package config

import (
	"strings"

	"github.com/caarlos0/env/v11"
)

type (
	// AppConfig holds all application configuration.
	AppConfig struct {
		DB          DB
		HTTPServer  HTTPServer
		Session     Session
		Environment Environment
	}

	// DB holds database specific values.
	DB struct {
		URL string `env:"DATABASE_URL"`
	}

	// HTTPServer holds HTTP server specific values.
	HTTPServer struct {
		Host string `env:"HOST" envDefault:"localhost"`
		Port string `env:"PORT" envDefault:"8080"`

		ReadTimeout  int `env:"READ_TIMEOUT_SECONDS" envDefault:"5"`
		WriteTimeout int `env:"WRITE_TIMEOUT_SECONDS" envDefault:"10"`
	}

	// Session holds session encryption configuration.
	Session struct {
		EncryptionKey string `env:"SESSION_ENCRYPTION_KEY"`
	}

	// Environment holds environment type.
	Environment struct {
		Type string `env:"ENVIRONMENT" envDefault:"local"`
	}
)

// Load loads the application configuration from environment variables.
func Load() (AppConfig, error) {
	return env.ParseAsWithOptions[AppConfig](env.Options{
		// Prefix: "PATCHWORKS_",
	})
}

func (e Environment) IsLocal() bool {
	return !strings.EqualFold(e.Type, "production")
}
