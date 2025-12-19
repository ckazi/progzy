package utils

import (
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	IsAdmin           bool   `json:"is_admin"`
	TwoFactorVerified bool   `json:"two_factor_verified"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	if hash == "" {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err == nil {
		return true
	}
	// Fallback for legacy/plaintext passwords if database still stores them unhashed.
	if subtle.ConstantTimeCompare([]byte(hash), []byte(password)) == 1 {
		log.Println("Warning: plaintext password detected, consider resetting this account's password.")
		return true
	}
	return false
}

func GenerateToken(userID int, username string, isAdmin bool, twoFactorVerified bool) (string, error) {
	return generateToken(userID, username, isAdmin, twoFactorVerified, 24*time.Hour)
}

func GenerateTempToken(userID int, username string, isAdmin bool) (string, error) {
	return generateToken(userID, username, isAdmin, false, 5*time.Minute)
}

func generateToken(userID int, username string, isAdmin bool, twoFactorVerified bool, ttl time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}

	claims := Claims{
		UserID:            userID,
		Username:          username,
		IsAdmin:           isAdmin,
		TwoFactorVerified: twoFactorVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
