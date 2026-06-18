package storage

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"url-shortener/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailAlreadyExists = errors.New("email already exists")

type UserStorage interface {
	CreateUser(ctx context.Context, email, passwordHash string) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

type SQLUserStorage struct {
	db *sql.DB
}

func NewSQLUserStorage(db *sql.DB) *SQLUserStorage {
	return &SQLUserStorage{db: db}
}

func (s *SQLUserStorage) CreateUser(ctx context.Context, email, passwordHash string) (int64, error){
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var id int64
	err := s.db.QueryRowContext(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id", email, passwordHash).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return 0, ErrEmailAlreadyExists
		}
		return 0, err
	}
	return id, nil
	
}

func (s *SQLUserStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRowContext(ctx, "SELECT id, email, password_hash, created_at FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows){
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
