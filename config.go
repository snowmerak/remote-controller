package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// Config holds the application configuration.
type Config struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	Port             string `json:"port"`
	DBPath           string `json:"db_path"`
	AllowedDirsRegex string `json:"allowed_dirs_regex"`
}

var (
	AppConfig         Config
	AllowedDirsRegexp *regexp.Regexp
)

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
	if AppConfig.AllowedDirsRegex == "" {
		AppConfig.AllowedDirsRegex = ".*"
	}

	var errCompile error
	AllowedDirsRegexp, errCompile = regexp.Compile(AppConfig.AllowedDirsRegex)
	if errCompile != nil {
		return fmt.Errorf("failed to compile allowed_dirs_regex: %w", errCompile)
	}

	return nil
}
