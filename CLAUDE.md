# Pipes - Project Instructions

This is a Go application following Herald's architecture patterns.

## Tech Stack

- **Language**: Go 1.24+
- **Database**: SQLite with direct SQL (no ORM)
- **Auth**: Indiko OAuth 2.0 server
- **Logging**: charmbracelet/log for structured logging
- **Frontend**: Go html/template + Vanilla JavaScript
- **Deployment**: Single static binary

## Development Commands

```bash
# Build the project
go build -o pipes .

# Run in development
./pipes serve -c config.yaml

# Run with hot reload (using a tool like air)
air

# Initialize config files
./pipes init

# Run tests
go test ./...
```

## Configuration

- **YAML Config** (config.yaml): All non-sensitive configuration
- **Environment** (.env): Secrets only (INDIKO_CLIENT_SECRET, SESSION_SECRET)
- YAML supports env var expansion: `${VAR}` syntax

See config.yaml.example and .env.example for templates.

## Architecture

Follow Herald's patterns:
- Clean separation of concerns (config/, store/, auth/, engine/, nodes/, web/)
- SQLite with WAL mode
- Structured logging with charm log
- Graceful shutdown with signal handling
- Session-based authentication with Indiko OAuth

## Code Style

- Use `gofmt` for formatting
- Structured logging: `logger.Info("message", "key", value)`
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Context propagation for cancellation
- Foreign key constraints in SQLite

## Design Aesthetic

**Neo-Brutalism** - Bold, geometric design:
- Space Grotesk font
- Hard borders (3-4px solid)
- Hard box shadows (no blur)
- High contrast colors
- Sharp, geometric shapes
- Color palette:
  - Primary: #2563eb (blue)
  - Secondary: #ff6b35 (orange)
  - Auth/Indiko: #AB4967 (pink)
  - Dark: #26242b (near-black)
  - Background: #f5f5f0 (warm off-white)
  - White: #fff
