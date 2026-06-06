package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/awnumar/memguard"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtKeyEnclave *memguard.Enclave
)

// InitJWTKey generates a random 32-byte signing key and stores it in a secure memguard Enclave.
func InitJWTKey() {
	jwtKeyEnclave = memguard.NewEnclaveRandom(32)
}

// GenerateToken signs a JWT for the specified username.
func GenerateToken(username string) (string, error) {
	// Retrieve signing key from enclave
	buf, err := jwtKeyEnclave.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open memguard enclave: %w", err)
	}
	defer buf.Destroy()

	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(buf.Bytes())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// JWTMiddleware validates the Bearer token in the Authorization header.
func JWTMiddleware(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format. Use 'Bearer <token>'",
		})
	}

	tokenString := parts[1]

	// Retrieve key from memguard for verification
	buf, err := jwtKeyEnclave.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server security error",
		})
	}
	defer buf.Destroy()

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return buf.Bytes(), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token claims",
		})
	}

	username, ok := claims["username"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username in token",
		})
	}

	c.Locals("username", username)
	return c.Next()
}
