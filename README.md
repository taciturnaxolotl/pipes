# Pipes

This is my interpretation of Yahoo Pipes from back in the day! It is designed to allow you to string together pipelines of data and do cool stuff with a modern Frutiger Aero aesthetic!

The canonical repo for this is hosted on tangled over at [`dunkirk.sh/pipes`](https://tangled.org/@dunkirk.sh/pipes)

## Features

- 🔐 **Passwordless Authentication** - Uses Indiko for OAuth 2.0 authentication with passkeys
- 🌊 **Visual Pipeline Builder** - Create data flows with an intuitive drag-and-drop interface
- ⚡ **Scheduled Execution** - Pipes run automatically on cron schedules
- 📊 **Data Sources** - RSS/Atom feeds and HTTP/REST APIs
- 🔄 **Transform Operations** - Filter, sort, limit, merge, dedupe, and extract data
- 🎨 **Neo-Brutalist Design** - Bold, geometric UI matching Indiko's aesthetic
- 👥 **Role-based Access** - User and admin roles powered by Indiko

## Tech Stack

- **Language**: Go 1.24+
- **Database**: SQLite with direct SQL
- **Auth**: [Indiko](https://github.com/taciturnaxolotl/indiko) OAuth 2.0 server
- **Frontend**: Go html/template + Vanilla JavaScript
- **Deployment**: Single static binary

## Installation

1. Clone the repository:

```bash
git clone https://github.com/taciturnaxolotl/pipes.git
cd pipes
```

2. Build the binary:

```bash
go build -o pipes .
```

3. Initialize configuration:

```bash
./pipes init
```

This creates a `config.yaml` file with sample configuration and a `.env.example` file for secrets.

Copy the example and add your secrets:

```bash
cp .env.example .env
# Edit .env with your actual secrets
```

Example `.env` file:

```env
# Pipes Secrets
# All other configuration is in config.yaml
# Copy this file to .env and fill in the secrets

# OAuth (Indiko)
INDIKO_CLIENT_SECRET=your_client_secret_here

# Session (generate with: openssl rand -base64 32)
SESSION_SECRET=your_random_secret_here
```

The database will be automatically created at `./pipes.db` on first run.

4. Set up Indiko OAuth:

Pipes uses auto-registration with Indiko, so you can start using it immediately! The client ID is just your app's URL (`http://localhost:3001`).

For production or to use role-based access control, ask your Indiko admin to pre-register your client with a client secret.

5. Start the server:

```bash
./pipes serve -c config.yaml
```

Or run without specifying a config file (uses environment variables from `.env`):

```bash
./pipes serve
```

Visit `http://localhost:3001` and sign in with your Indiko account!

## Configuration

Pipes uses a two-file configuration approach (just like Herald):

### YAML Config File (config.yaml)

Contains all non-sensitive configuration:

```bash
./pipes init              # Creates config.yaml and .env.example
./pipes serve -c config.yaml
```

Example `config.yaml`:
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

Contains **only secrets** (never commit this file):

```env
# OAuth (Indiko)
INDIKO_CLIENT_SECRET=your_client_secret_here

# Session (generate with: openssl rand -base64 32)
SESSION_SECRET=your_random_secret_here
```

YAML config supports environment variable expansion using `${VAR}` syntax. Variables are loaded from `.env` file and can be overridden by system environment variables.

**Configuration precedence:** Environment variables > YAML config > defaults

### Log Levels

Set `LOG_LEVEL` (or `log_level` in YAML) to:
- `debug` - Verbose output for troubleshooting
- `info` - Standard operational messages (default)
- `warn` - Warning messages
- `error` - Error messages only
- `fatal` - Fatal errors (exits immediately)

Example structured logging output:
```
2026/01/10 10:24:05 INFO starting pipes host=localhost port=3001 db_path=pipes.db
2026/01/10 10:24:05 INFO user authenticated name="John Doe" email="john@example.com"
```

## Architecture

Pipes follows Herald's clean architecture patterns:

```
pipes/
├── main.go                 # CLI entry point
├── config/                 # Configuration management
├── store/                  # Database operations
├── auth/                   # OAuth 2.0 client & session management
├── engine/                 # Pipeline executor & scheduler
├── nodes/                  # Node type definitions
│   ├── sources/           # RSS, HTTP API sources
│   └── transforms/        # Filter, sort, limit operations
└── web/                   # HTTP server & handlers
    └── templates/         # HTML templates
```

## OAuth Flow

1. User clicks "Sign in with Indiko"
2. Redirect to Indiko authorization endpoint with PKCE
3. User authenticates with passkey on Indiko
4. User approves scopes (profile, email)
5. Indiko redirects back with authorization code
6. Exchange code for access + refresh tokens
7. Create/update user in local database
8. Create session with 30-day cookie

## Pipeline Execution

Pipelines are executed using topological sort (Kahn's algorithm):

1. Parse pipe configuration (nodes + connections)
2. Build dependency graph
3. Execute nodes in order, passing data between them
4. Log execution progress
5. Store results in database

The scheduler runs every minute, checking for pipes that need to execute based on their cron schedules.

## Available Node Types

**Sources:**
- RSS Feed - Fetch items from RSS/Atom feeds
- HTTP API - Fetch JSON data from REST APIs (coming soon)

**Transforms:**
- Filter - Filter items based on field conditions
- Sort - Sort items by field values
- Limit - Limit the number of output items
- Merge - Combine multiple data sources (coming soon)
- Dedupe - Remove duplicate items (coming soon)
- Extract - Transform/extract fields (coming soon)

## Development

Build and run:

```bash
go build -o pipes .
./pipes serve
```

The database schema is automatically created on first run.

<p align="center">
    <img src="https://raw.githubusercontent.com/taciturnaxolotl/carriage/main/.github/images/line-break.svg" />
</p>

<p align="center">
    <i><code>&copy 2025-present <a href="https://dunkirk.sh">Kieran Klukas</a></code></i>
</p>

<p align="center">
    <a href="https://tangled.org/dunkirk.sh/indiko/blob/main/LICENSE.md"><img src="https://img.shields.io/static/v1.svg?style=for-the-badge&label=License&message=O'Saasy&logoColor=d9e0ee&colorA=363a4f&colorB=b7bdf8"/></a>
</p>

