package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Server
	Origin   string `yaml:"origin"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Env      string `yaml:"env"`
	LogLevel string `yaml:"log_level"`

	// Database
	DatabasePath string `yaml:"db_path"`

	// OAuth (Indiko)
	IndikoURL          string `yaml:"indiko_url"`
	IndikoClientID     string `yaml:"indiko_client_id"`
	IndikoClientSecret string `yaml:"indiko_client_secret"`
	OAuthCallbackURL   string `yaml:"oauth_callback_url"`

	// Session
	SessionSecret     string `yaml:"session_secret"`
	SessionCookieName string `yaml:"session_cookie_name"`
}

// Default returns a Config with sensible defaults
func Default() *Config {
	return &Config{
		Origin:            "http://localhost:3001",
		Host:              "localhost",
		Port:              3001,
		Env:               "development",
		LogLevel:          "info",
		DatabasePath:      "pipes.db",
		IndikoURL:         "http://localhost:3000",
		OAuthCallbackURL:  "http://localhost:3001/auth/callback",
		SessionCookieName: "pipes_session",
	}
}

// Load loads configuration from YAML file (if provided) and environment variables
func Load(path string) (*Config, error) {
	cfg := Default()

	// Load .env file if it exists (silently ignore if not found)
	if envPath := findEnvFile(path); envPath != "" {
		_ = godotenv.Load(envPath)
	}

	// Load from YAML config file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Expand environment variables in YAML (e.g., ${DATABASE_PATH})
		expanded := os.Expand(string(data), func(key string) string {
			return os.Getenv(key)
		})

		if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.SessionSecret == "" {
		return fmt.Errorf("session_secret is required (set SESSION_SECRET env var)")
	}

	if c.IndikoClientID == "" {
		return fmt.Errorf("indiko_client_id is required (set INDIKO_CLIENT_ID env var)")
	}

	if c.IndikoURL == "" {
		return fmt.Errorf("indiko_url is required (set INDIKO_URL env var)")
	}

	return nil
}

// findEnvFile looks for .env file in the config file's directory or current directory
func findEnvFile(configPath string) string {
	// If config path provided, look in its directory
	if configPath != "" {
		dir := filepath.Dir(configPath)
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Look in current directory
	if _, err := os.Stat(".env"); err == nil {
		return ".env"
	}

	return ""
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("ORIGIN"); v != "" {
		cfg.Origin = v
	}
	if v := os.Getenv("HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("NODE_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("DATABASE_PATH"); v != "" {
		cfg.DatabasePath = v
	}
	if v := os.Getenv("INDIKO_URL"); v != "" {
		cfg.IndikoURL = v
	}
	if v := os.Getenv("INDIKO_CLIENT_ID"); v != "" {
		cfg.IndikoClientID = v
	}
	if v := os.Getenv("INDIKO_CLIENT_SECRET"); v != "" {
		cfg.IndikoClientSecret = v
	}
	if v := os.Getenv("OAUTH_CALLBACK_URL"); v != "" {
		cfg.OAuthCallbackURL = v
	}
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		cfg.SessionSecret = v
	}
	if v := os.Getenv("SESSION_COOKIE_NAME"); v != "" {
		cfg.SessionCookieName = v
	}
}
