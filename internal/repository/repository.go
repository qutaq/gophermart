package repository

import (
	"context"
	"errors"

	"github.com/qutaq/gophermart/internal/domain"
)

var (
	ErrConflict = errors.New("repository: conflict")
	ErrNotFound = errors.New("repository: not found")
)

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (domain.User, error)
	ByLogin(ctx context.Context, login string) (domain.User, error)
}

type OrderRepository interface {
	Create(ctx context.Context, userID int64, number string) error
	ByNumber(ctx context.Context, number string) (domain.Order, error)
	ByUser(ctx context.Context, userID int64) ([]domain.Order, error)
}
