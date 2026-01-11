# Crush Memory - Pipes Project

## User Preferences

- Follow Herald's Go architecture patterns
- Use direct SQL for all database operations (no ORM)
- Follow neo-brutalist design aesthetic (blue/orange for app, pink only for auth)
- Use Indiko for authentication (OAuth 2.0 with PKCE)
- Run on port 3001 (Indiko runs on 3000)
- Use charmbracelet/log for structured logging

## Architecture Patterns

### Authentication Flow
- OAuth 2.0 client that uses Indiko for authentication
- PKCE flow with code verifier/challenge (required by IndieAuth spec)
- Auto-registration with Indiko (client ID = app URL)
- Session-based auth with 30-day cookies
- Automatic token refresh using refresh tokens

### Database (SQLite)
- Direct SQL queries (no ORM like Kysely or GORM)
- SQLite with WAL mode for concurrency
- Foreign key constraints enabled
- Schema created automatically on startup

### Project Structure
```
pipes/
├── main.go                     # Entry point, CLI setup
├── go.mod                      # Dependencies
├── config/
│   ├── app.go                 # Config struct & loading (like Herald)
│   └── validate.go            # Config validation
├── store/
│   ├── db.go                  # SQLite setup & schema
│   ├── users.go               # User operations
│   ├── pipes.go               # Pipe CRUD
│   ├── executions.go          # Execution history
│   └── cache.go               # Source cache operations
├── auth/
│   ├── oauth.go               # OAuth 2.0 client (Indiko)
│   ├── session.go             # Session management
│   └── middleware.go          # Auth middleware
├── engine/
│   ├── executor.go            # Pipeline execution engine
│   ├── scheduler.go           # Cron-based scheduler (Herald pattern)
│   └── registry.go            # Node type registry
├── nodes/
│   ├── node.go                # Node interface
│   ├── sources/
│   │   ├── rss.go            # RSS/Atom source
│   │   └── http.go           # HTTP API source
│   └── transforms/
│       ├── filter.go         # Filter transform
│       ├── sort.go           # Sort transform
│       ├── limit.go          # Limit transform
│       ├── merge.go          # Merge sources
│       ├── dedupe.go         # Deduplicate items
│       └── extract.go        # Extract/transform fields
└── web/
    ├── server.go             # HTTP server setup
    ├── handlers.go           # Route handlers
    ├── api.go                # JSON API endpoints
    └── templates/
        ├── layout.html       # Base layout
        ├── index.html        # Landing page
        ├── dashboard.html    # User dashboard
        ├── editor.html       # Visual editor
        └── style.css         # Frutiger Aero styles
```

### Database Schema
- **users**: id, indiko_sub, username, name, email, photo, url, role, created_at, updated_at
  - `indiko_sub` is the "sub" field from Indiko's userinfo endpoint
  - `role` is either "user" or "admin" (synced from Indiko)
- **sessions**: id, user_id, access_token, refresh_token, expires_at, created_at
  - 30-day sessions with automatic token refresh
- **pipes**: id, user_id, name, description, config (JSON), is_public, created_at, updated_at
  - `config` is JSON: {version, nodes[], connections[], settings}
- **scheduled_jobs**: id, pipe_id, cron_expression, next_run_at, last_run_at, enabled, created_at, updated_at
- **pipe_executions**: id, pipe_id, status, trigger_type, started_at, completed_at, duration_ms, items_processed, error_message, metadata
- **execution_logs**: id, execution_id, node_id, level, message, timestamp, metadata
- **source_cache**: id, pipe_id, node_id, cache_key, data, etag, last_modified, expires_at, created_at

### OAuth Configuration
- **Client ID**: App's URL (e.g., `http://localhost:3001`)
- **Client Secret**: Optional, for pre-registered clients (stored in .env)
- **Scopes**: `profile email`
- **PKCE**: Required (S256 code challenge method)
- **Callback URL**: `{ORIGIN}/auth/callback`

## Configuration

### YAML Config (config.yaml)
Contains all non-sensitive configuration:
```yaml
# Server settings
host: localhost
port: 3001
origin: http://localhost:3001
env: development
log_level: info  # debug, info, warn, error, fatal

# Database
db_path: pipes.db

# OAuth (Indiko)
indiko_url: http://localhost:3000
indiko_client_id: http://localhost:3001
indiko_client_secret: ${INDIKO_CLIENT_SECRET}  # Loaded from .env
oauth_callback_url: http://localhost:3001/auth/callback

# Session
session_secret: ${SESSION_SECRET}  # Loaded from .env
session_cookie_name: pipes_session
```

### Environment Variables (.env)
Contains **only secrets**:
```env
# OAuth (Indiko)
INDIKO_CLIENT_SECRET=your_client_secret_here

# Session (generate with: openssl rand -base64 32)
SESSION_SECRET=your_random_secret_here
```

## Routes

### Public
- `GET /` - Landing page (redirects to /dashboard if authenticated)
- `GET /auth/login` - Start OAuth flow
- `GET /auth/callback` - OAuth callback handler

### Authenticated
- `GET /dashboard` - User dashboard (requires auth)
- `GET /pipes/:id/edit` - Visual editor (requires auth)
- `POST /auth/logout` - End session
- `GET /api/me` - Get current user info
- `GET /api/pipes` - List user's pipes
- `GET /api/pipes/:id` - Get pipe config
- `POST /api/pipes` - Create pipe
- `PUT /api/pipes/:id` - Update pipe
- `DELETE /api/pipes/:id` - Delete pipe
- `POST /api/pipes/:id/execute` - Execute pipe manually
- `GET /api/pipes/:id/executions` - Execution history
- `GET /api/executions/:id/logs` - Execution logs
- `GET /api/node-types` - Available node types

## Code Style

- Use `gofmt` for formatting
- Structured logging with charmbracelet/log: `logger.Info("message", "key", value)`
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Context propagation for cancellation
- Session cookies named `pipes_session`
- Authorization header: `Bearer {token}`

## Design Aesthetic

**Neo-Brutalism** - Bold, geometric design:
- Space Grotesk font family
- Hard borders (3-4px solid #26242b)
- Hard box shadows (6-12px offset, no blur)
- High contrast, sharp edges
- Uppercase text, tight letter-spacing
- Color palette:
  - **App colors**:
    - Primary: #2563eb (blue) - used for headings, secondary buttons
    - Secondary: #ff6b35 (orange) - used for primary buttons, accents
    - Background: #f5f5f0 (warm off-white)
    - Dark: #26242b (near-black)
    - White: #fff
  - **Auth/Indiko colors** (login, error pages):
    - Auth primary: #AB4967 (muted pink)
    - Auth text: #fff (white - for text on pink buttons)

## Commands

```bash
go build -o pipes .     # Build
./pipes serve           # Run server
./pipes init            # Initialize config files
./pipes help            # Show help
./pipes version         # Show version
```
