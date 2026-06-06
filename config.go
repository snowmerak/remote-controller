package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Port     string `json:"port"`
	DBPath   string `json:"db_path"`
}

var AppConfig Config

// LoadConfig reads configuration from the given file path.
func LoadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&AppConfig); err != nil {
		return fmt.Errorf("failed to decode config JSON: %w", err)
	}

	if AppConfig.Username == "" || AppConfig.Password == "" {
		return fmt.Errorf("username and password must not be empty in config")
	}
	if AppConfig.Port == "" {
		AppConfig.Port = ":8080"
	}
	if AppConfig.DBPath == "" {
		AppConfig.DBPath = "history.db"
	}

	return nil
}
