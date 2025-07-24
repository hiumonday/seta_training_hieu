package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Claims struct that matches Node.js token payload exactly
type Claims struct {
	UserID uuid.UUID `json:"userId"` // Match Node.js payload format
	jwt.RegisteredClaims
}

// ValidateToken validates JWT token from Node.js service
func ValidateToken(tokenString string) (*Claims, error) {
	// Use ACCESS_TOKEN_SECRET to match Node.js service
	secret := os.Getenv("ACCESS_TOKEN_SECRET")
	if secret == "" {
		return nil, errors.New("ACCESS_TOKEN_SECRET not set")
	}

	// Debug: Print secret to verify it matches
	fmt.Printf("DEBUG: Using ACCESS_TOKEN_SECRET: %s\n", secret[:10]+"...")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		fmt.Printf("DEBUG: JWT parse error: %v\n", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		fmt.Printf("DEBUG: Invalid token claims or token not valid\n")
		return nil, errors.New("invalid token")
	}

	fmt.Printf("DEBUG: Successfully parsed token for user: %s\n", claims.UserID)
	return claims, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
