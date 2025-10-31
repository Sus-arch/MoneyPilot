package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	DB        *sql.DB
	JWTSecret []byte
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthService(db *sql.DB, secret string) *AuthService {
	return &AuthService{
		DB:        db,
		JWTSecret: []byte(secret),
	}
}

func (s *AuthService) Authenticate(email, password string) (string, error) {
	var id int
	var dbPass string

	err := s.DB.QueryRow(`SELECT id, password_hash FROM users WHERE email=$1`, email).Scan(&id, &dbPass)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if dbPass != password {
		return "", errors.New("invalid credentials")
	}

	claims := Claims{
		UserID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.JWTSecret)
}
