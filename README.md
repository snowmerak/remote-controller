# remote-controller

Remote Controller is a secure, lightweight Go API built with **Fiber v3** that executes command-line sessions (`agx` and `grok`) in specific local directories and tracks command execution history in a CGO-free **SQLite** database.

## Features

- **Directory-Based Sessions**: Map local directory paths to aliases and CLI services (`agx` or `grok`).
- **Path Validation**: Session directories and traversal paths are verified against a regular expression pattern specified in the configuration file.
- **Directory Traversal**: Navigate directories matching validation criteria using filesystem exploration endpoints.
- **Secure Authentication**: JWT token-based API endpoints.
- **Transient Session Signing Key**: JWT HMAC signing key is randomly generated on startup and kept locked in memory using **memguard**; all tokens invalidate when the server restarts.
- **Subprocess Command execution**: Run commands contextually inside mapped workspaces:
  - `agx "prompt"` (includes auto-initialization via `agx init --auto`).
  - `grok -c -p "prompt"`.
- **Query History Logging**: Execution outputs, statuses, errors, and timestamps are persisted in SQLite with pagination support.

## Configuration

Place a `config.json` file in the root directory:

```json
{
  "username": "admin",
  "password": "password123",
  "port": ":8080",
  "db_path": "history.db",
  "allowed_dirs_regex": "^C:/Users/snowm/Projects/.*$"
}
```

## Running the Server

Ensure dependencies are installed and run:

```bash
go run .
```

## API Endpoints

### Authentication
- `POST /api/login`: Logs in with configuration credentials. Returns a JWT token.

### Sessions & Traversal (Requires Authentication)
- `GET /api/explore`: Explore filesystem directories.
  - Query parameters: `path` (absolute path to list).
- `POST /api/sessions`: Register or update a session.
  - Body: `{ "alias": "name", "directory": "/path/to/dir", "service": "agx" }`
- `GET /api/sessions`: List registered sessions with pagination.
  - Query parameters: `page` (default `1`), `limit` (default `10`).
- `DELETE /api/sessions/:alias`: Deregister a session.

### Execution & History (Requires Authentication)
- `POST /api/query`: Runs a prompt in the session's workspace.
  - Body: `{ "alias": "name", "prompt": "prompt description" }`
- `GET /api/history`: Get paginated log entries, optionally filtered by alias.
  - Query parameters: `alias` (optional), `page` (default `1`), `limit` (default `10`).

## Security Architecture

- **memguard**: Wipes and securely seals the memory holding the JWT token secret key to prevent leakages via core dumps or swap spaces.
- **Graceful Shutdown**: Automatically catches interrupt signals, closes database connections, shuts down the HTTP listener, and purges memguard enclaves before process exit.
