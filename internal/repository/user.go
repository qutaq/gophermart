package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qutaq/gophermart/internal/domain"
	"github.com/qutaq/gophermart/internal/pgerr"
)

var _ UserRepository = (*userRepository)(nil)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, login, passwordHash string) (domain.User, error) {
	const query = `
		INSERT INTO users (login, password)
		VALUES ($1, $2)
		RETURNING id, login, password
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, login, passwordHash).
		Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		if pgerr.UniqueViolation(err) {
			return domain.User{}, ErrConflict
		}
		return domain.User{}, fmt.Errorf("user repository: create: %w", err)
	}
	return user, nil
}

func (r *userRepository) ByLogin(ctx context.Context, login string) (domain.User, error) {
	const query = `
		SELECT id, login, password
		FROM users
		WHERE login = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, login).
		Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, fmt.Errorf("user repository: by login: %w", err)
	}
	return user, nil
}

func (r *userRepository) Balance(ctx context.Context, userID int64) (domain.Kopecks, domain.Kopecks, error) {
	const query = `
		SELECT balance, withdrawn
		FROM users
		WHERE id = $1
	`

	var current, withdrawn domain.Kopecks
	err := r.db.QueryRow(ctx, query, userID).Scan(&current, &withdrawn)
	if err != nil {
		return 0, 0, fmt.Errorf("user repository: balance: %w", err)
	}
	return current, withdrawn, nil
}

func (r *userRepository) Withdraw(ctx context.Context, userID int64, orderNumber string, sum domain.Kopecks) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user repository: withdraw: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance domain.Kopecks
	err = tx.QueryRow(ctx, `
		SELECT balance FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("user repository: withdraw: select: %w", err)
	}

	if balance < sum {
		return ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx, `
		UPDATE users SET balance = balance - $1, withdrawn = withdrawn + $1 WHERE id = $2
	`, sum, userID)
	if err != nil {
		return fmt.Errorf("user repository: withdraw: update: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)
	`, userID, orderNumber, sum)
	if err != nil {
		return fmt.Errorf("user repository: withdraw: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("user repository: withdraw: commit: %w", err)
	}
	return nil
}
