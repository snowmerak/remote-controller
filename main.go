package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/awnumar/memguard"
	"github.com/gofiber/fiber/v3"
)

func main() {
	// Initialize memguard and ensure memory is purged on exit
	memguard.CatchInterrupt()
	defer memguard.Purge()

	// Initialize the random JWT secret key in memory enclave
	InitJWTKey()

	// Load config.json file
	configPath := "config.json"
	if err := LoadConfig(configPath); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize SQLite database connection
	if err := InitDB(AppConfig.DBPath); err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	// Initialize Fiber v3 application
	app := fiber.New()

	// Public routes
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendFile("./static/index.html")
	})
	app.Post("/api/login", HandleLogin)

	// Authenticated routes group
	api := app.Group("/api", JWTMiddleware)
	api.Post("/sessions", HandleCreateSession)
	api.Post("/sessions/chat", HandleCreateChatSession)
	api.Get("/sessions", HandleListSessions)
	api.Delete("/sessions/:alias", HandleDeleteSession)
	api.Post("/query", HandleQuery)
	api.Get("/history", HandleGetHistory)
	api.Get("/explore", HandleExplore)

	// Start server on a background goroutine to support deferred cleanups
	go func() {
		log.Printf("Server starting on %s", AppConfig.Port)
		if err := app.Listen(AppConfig.Port); err != nil {
			log.Printf("Server execution error: %v", err)
		}
	}()

	// Graceful shutdown listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error shutting down: %v", err)
	}
}
