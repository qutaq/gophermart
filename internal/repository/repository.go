package repository

import (
	"context"
	"errors"

	"github.com/qutaq/gophermart/internal/domain"
)

var (
	ErrConflict         = errors.New("repository: conflict")
	ErrNotFound         = errors.New("repository: not found")
	ErrInsufficientFunds = errors.New("repository: insufficient funds")
)

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (domain.User, error)
	ByLogin(ctx context.Context, login string) (domain.User, error)
	Balance(ctx context.Context, userID int64) (current, withdrawn domain.Kopecks, err error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum domain.Kopecks) error
}

type WithdrawalRepository interface {
	ByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
}

type OrderRepository interface {
	Create(ctx context.Context, userID int64, number string) error
	ByNumber(ctx context.Context, number string) (domain.Order, error)
	ByUser(ctx context.Context, userID int64) ([]domain.Order, error)
	Pending(ctx context.Context, limit int) ([]domain.Order, error)
	ApplyAccrual(ctx context.Context, number, status string, accrual *domain.Kopecks, userID int64) error
}
