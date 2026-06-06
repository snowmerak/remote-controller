package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// HandleLogin authenticates users and issues JWT tokens.
func HandleLogin(c fiber.Ctx) error {
	req := new(LoginRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Username != AppConfig.Username || req.Password != AppConfig.Password {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
	}

	token, err := GenerateToken(req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(LoginResponse{Token: token})
}

type CreateSessionRequest struct {
	Alias     string `json:"alias"`
	Directory string `json:"directory"`
	Service   string `json:"service"` // "agx" or "grok"
}

// HandleCreateSession registers a new workspace session and runs service-specific initialization.
func HandleCreateSession(c fiber.Ctx) error {
	req := new(CreateSessionRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Alias == "" || req.Directory == "" || req.Service == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "alias, directory, and service are required fields"})
	}

	if req.Service != "agx" && req.Service != "grok" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "service must be either 'agx' or 'grok'"})
	}

	normalizedDir := NormalizePath(req.Directory)
	if !IsValidPath(normalizedDir) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Directory path is not allowed by configuration"})
	}

	s := Session{
		Alias:     req.Alias,
		Directory: normalizedDir,
		Service:   req.Service,
	}

	// Initialize the session (runs agx init if needed)
	if err := InitDirectorySession(s); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if err := SaveSession(s.Alias, s.Directory, s.Service); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save session to database"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"session": s,
	})
}

// HandleListSessions returns a paginated list of all active sessions.
func HandleListSessions(c fiber.Ctx) error {
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	list, total, err := ListSessions(page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve sessions"})
	}

	return c.JSON(fiber.Map{
		"sessions": list,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// HandleDeleteSession closes and removes a session by alias.
func HandleDeleteSession(c fiber.Ctx) error {
	alias := c.Params("alias")
	if alias == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Alias parameter is required"})
	}

	// First verify it exists
	_, err := GetSession(alias)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Session not found"})
	}

	if err := DeleteSession(alias); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete session"})
	}

	return c.JSON(fiber.Map{"status": "success"})
}

type QueryRequest struct {
	Alias  string `json:"alias"`
	Prompt string `json:"prompt"`
}

// HandleQuery executes a query in the directory associated with a session and saves logs to SQLite.
func HandleQuery(c fiber.Ctx) error {
	req := new(QueryRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Alias == "" || req.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "alias and prompt are required fields"})
	}

	s, err := GetSession(req.Alias)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Session not found"})
	}

	output, queryErr := RunQuery(s, req.Prompt)

	// Save history log to SQLite database
	if dbErr := SaveHistory(s.Alias, req.Prompt, output, queryErr); dbErr != nil {
		fmt.Printf("Warning: failed to save history log: %v\n", dbErr)
	}

	if queryErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  queryErr.Error(),
			"output": output,
		})
	}

	return c.JSON(fiber.Map{
		"output": output,
	})
}

// HandleGetHistory returns a paginated history list, optionally filtered by session alias.
func HandleGetHistory(c fiber.Ctx) error {
	alias := c.Query("alias", "")
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	list, total, err := GetHistory(alias, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve history"})
	}

	return c.JSON(fiber.Map{
		"history": list,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

type ExploreItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// HandleExplore lists contents of a directory if validated against allowed_dirs_regex.
func HandleExplore(c fiber.Ctx) error {
	path := c.Query("path", "")
	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "path parameter is required"})
	}

	normalizedPath := NormalizePath(path)
	if !IsValidPath(normalizedPath) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access to path is not allowed by configuration"})
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Path does not exist"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to stat path: %v", err)})
	}

	if !info.IsDir() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Path is not a directory"})
	}

	entries, err := os.ReadDir(normalizedPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to read directory: %v", err)})
	}

	var items []ExploreItem
	for _, entry := range entries {
		items = append(items, ExploreItem{
			Name:  entry.Name(),
			Path:  NormalizePath(filepath.Join(normalizedPath, entry.Name())),
			IsDir: entry.IsDir(),
		})
	}

	return c.JSON(fiber.Map{
		"current_path": normalizedPath,
		"items":        items,
	})
}
