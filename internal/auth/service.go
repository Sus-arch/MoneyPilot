package auth

import (
	"MoneyPilot/internal/storage"
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

func (s *AuthService) Authenticate(email, password, bank string) (string, error) {
	// Use repository helper to resolve user within the context of the requested bank.
	repo := storage.NewRepository(s.DB)
	user, err := repo.GetUserByClientIDAndBank(email, bank)
	// log.Println(user)
	if err != nil {
		// hide detailed DB errors from caller
		return "", errors.New("invalid credentials")
	}

	// Compare password using bcrypt to support hashed passwords in DB.
	if password != user.PasswordHash {
		return "", errors.New("invalid credentials")
	}

	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.JWTSecret)
}

func (s *AuthService) AuthenticateConsent(email, bank string) (bool, error) {
	rep := storage.NewRepository(s.DB)

	cons, err := rep.GetValidAccountConsentsByEmailAndBank(email, bank)

	if err != nil {
		return false, err
	}
	return len(cons) > 0, nil
}
