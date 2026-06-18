package service

import (
	"context"
	"errors"
	"time"
	"url-shortener/internal/storage"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ( ErrInvalidCredentials = errors.New("invalid credentials") )

type AuthService struct {
	storage storage.UserStorage
	jwtSecret []byte
}

func NewAuthService(storage storage.UserStorage, jwtSecret []byte) *AuthService {
	return &AuthService{
		storage: storage,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, email, password string)  error {
	if err := ctx.Err(); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.storage.CreateUser(ctx, email, string(hash))
	return err
}

func (s *AuthService) LoginUser(ctx context.Context, email, password string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if id, ok := claims["user_id"].(float64); ok {
			return int64(id), nil
		}
	}
	return 0, ErrInvalidCredentials
}