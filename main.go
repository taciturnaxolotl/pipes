package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/kierank/pipes/config"
	"github.com/kierank/pipes/engine"
	"github.com/kierank/pipes/store"
	"github.com/kierank/pipes/web"
)

var (
	version    = "dev"
	commitHash = "dev"
	logger     *log.Logger
)

func main() {
	// Initialize logger with default level
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Level:           log.InfoLevel,
	})

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		configPath := ""
		// Check for -c or --config flag
		for i := 2; i < len(os.Args); i++ {
			if (os.Args[i] == "-c" || os.Args[i] == "--config") && i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				break
			}
		}
		serve(configPath)
	case "init":
		initConfig()
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("pipes %s (%s)\n", version, commitHash)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Pipes - Visual data pipeline builder")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pipes <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  serve              Start the server")
	fmt.Println("  init [path]        Create a sample config file (default: config.yaml)")
	fmt.Println("  version            Show version information")
	fmt.Println("  help               Show this help message")
	fmt.Println()
	fmt.Println("Serve Flags:")
	fmt.Println("  -c, --config PATH  Path to config file (optional, uses .env if not specified)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pipes init")
	fmt.Println("  pipes serve -c config.yaml")
	fmt.Println("  pipes serve                    # Uses .env file")
	fmt.Println()
}

func serve(configPath string) {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal("failed to load config", "error", err)
	}

	// Set log level from config
	level := parseLogLevel(cfg.LogLevel)
	logger.SetLevel(level)

	logger.Info("starting pipes",
		"host", cfg.Host,
		"port", cfg.Port,
		"db_path", cfg.DatabasePath,
		"log_level", cfg.LogLevel,
	)

	// Initialize database
	db, err := store.New(cfg.DatabasePath)
	if err != nil {
		logger.Fatal("failed to initialize database", "error", err)
	}
	defer db.Close()

	logger.Info("database initialized successfully")

	// Initialize scheduler
	scheduler := engine.NewScheduler(db, logger)
	scheduler.Start()
	defer scheduler.Stop()

	logger.Info("scheduler started")

	// Initialize web server
	server := web.NewServer(cfg, db, logger)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting server", "address", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		logger.Info("shutting down gracefully...")
	case err := <-serverErr:
		logger.Fatal("server error", "error", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("shutdown complete")
}

func initConfig() {
	configPath := "config.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file already exists at %s\n", configPath)
		fmt.Println("Remove it first or specify a different path:")
		fmt.Printf("  pipes init %s.new\n", configPath)
		os.Exit(1)
	}

	secret, err := generateSecret()
	if err != nil {
		logger.Fatal("failed to generate secret", "error", err)
	}

	configContent := `# Pipes Configuration
# See https://github.com/yourusername/pipes for documentation

# Server settings
host: localhost
port: 3001
origin: http://localhost:3001
env: development
log_level: info  # debug, info, warn, error, fatal

# Database
db_path: pipes.db

# OAuth (Indiko)
# Set these environment variables or replace with actual values:
indiko_url: ${INDIKO_URL}
indiko_client_id: ${INDIKO_CLIENT_ID}
indiko_client_secret: ${INDIKO_CLIENT_SECRET}
oauth_callback_url: http://localhost:3001/auth/callback

# Session
session_secret: ` + secret + `
session_cookie_name: pipes_session
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		logger.Fatal("failed to write config", "error", err)
	}

	fmt.Printf("✓ Config file created at %s\n\n", configPath)
	fmt.Println("Next steps:")
	fmt.Println("  1. Set your Indiko OAuth environment variables:")
	fmt.Println("     export INDIKO_URL=http://localhost:3000")
	fmt.Println("     export INDIKO_CLIENT_ID=http://localhost:3001")
	fmt.Println()
	fmt.Println("  2. Or edit the config file directly to replace ${VAR} placeholders")
	fmt.Println()
	fmt.Println("  3. Start the server:")
	fmt.Printf("     pipes serve -c %s\n", configPath)
	fmt.Println()
	fmt.Println("  Or use environment variables with a .env file instead:")
	fmt.Println("     cp .env.example .env")
	fmt.Println("     pipes serve")
}

func generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func parseLogLevel(levelStr string) log.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn", "warning":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}
