package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type Session struct {
	Alias     string    `json:"alias"`
	Directory string    `json:"directory"`
	Service   string    `json:"service"`
	CreatedAt time.Time `json:"created_at"`
}

type HistoryItem struct {
	ID           int64     `json:"id"`
	SessionAlias string    `json:"session_alias"`
	Prompt       string    `json:"prompt"`
	Response     string    `json:"response"`
	Error        *string   `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
}

// InitDB initializes the SQLite database connection and sets up tables.
func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Enable foreign key support
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			alias TEXT PRIMARY KEY,
			directory TEXT NOT NULL,
			service TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_alias TEXT NOT NULL,
			prompt TEXT NOT NULL,
			response TEXT NOT NULL,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_alias) REFERENCES sessions(alias) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return nil
}

// SaveSession registers or updates a session mapping in SQLite.
func SaveSession(alias, directory, service string) error {
	query := `INSERT INTO sessions (alias, directory, service, created_at)
			  VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			  ON CONFLICT(alias) DO UPDATE SET
			  directory=excluded.directory,
			  service=excluded.service;`
	_, err := db.Exec(query, alias, directory, service)
	return err
}

// DeleteSession removes a session from SQLite, cascading to delete related history.
func DeleteSession(alias string) error {
	query := `DELETE FROM sessions WHERE alias = ?`
	_, err := db.Exec(query, alias)
	return err
}

// GetSession retrieves a single session by its alias.
func GetSession(alias string) (Session, error) {
	query := `SELECT alias, directory, service, created_at FROM sessions WHERE alias = ?`
	row := db.QueryRow(query, alias)

	var s Session
	var createdAtStr string
	err := row.Scan(&s.Alias, &s.Directory, &s.Service, &createdAtStr)
	if err != nil {
		return s, err
	}

	s.CreatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", createdAtStr, time.UTC)
	if s.CreatedAt.IsZero() {
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	}

	return s, nil
}

// ListSessions retrieves a paginated list of sessions.
func ListSessions(page, limit int) ([]Session, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.Query("SELECT alias, directory, service, created_at FROM sessions ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Session
	for rows.Next() {
		var s Session
		var createdAtStr string
		if err := rows.Scan(&s.Alias, &s.Directory, &s.Service, &createdAtStr); err != nil {
			return nil, 0, err
		}
		s.CreatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", createdAtStr, time.UTC)
		if s.CreatedAt.IsZero() {
			s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		list = append(list, s)
	}

	return list, total, nil
}

// SaveHistory inserts a prompt/response log into the history table.
func SaveHistory(alias, prompt, response string, executionErr error) error {
	var errStr *string
	if executionErr != nil {
		s := executionErr.Error()
		errStr = &s
	}

	query := `INSERT INTO history (session_alias, prompt, response, error, created_at)
			  VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := db.Exec(query, alias, prompt, response, errStr)
	return err
}

// GetHistory retrieves a paginated query history list, optionally filtered by alias.
func GetHistory(alias string, page, limit int) ([]HistoryItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	var countQuery string
	var selectQuery string
	var args []interface{}

	if alias != "" {
		countQuery = "SELECT COUNT(*) FROM history WHERE session_alias = ?"
		selectQuery = "SELECT id, session_alias, prompt, response, error, created_at FROM history WHERE session_alias = ? ORDER BY created_at DESC LIMIT ? OFFSET ?"
		args = append(args, alias)
	} else {
		countQuery = "SELECT COUNT(*) FROM history"
		selectQuery = "SELECT id, session_alias, prompt, response, error, created_at FROM history ORDER BY created_at DESC LIMIT ? OFFSET ?"
	}

	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	selectArgs := append(args, limit, offset)
	rows, err := db.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []HistoryItem
	for rows.Next() {
		var h HistoryItem
		var createdAtStr string
		if err := rows.Scan(&h.ID, &h.SessionAlias, &h.Prompt, &h.Response, &h.Error, &createdAtStr); err != nil {
			return nil, 0, err
		}
		h.CreatedAt, _ = time.ParseInLocation("2006-01-02 15:04:05", createdAtStr, time.UTC)
		if h.CreatedAt.IsZero() {
			h.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		list = append(list, h)
	}

	return list, total, nil
}
