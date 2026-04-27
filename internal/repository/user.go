package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qutaq/gophermart/internal/domain"
)

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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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
