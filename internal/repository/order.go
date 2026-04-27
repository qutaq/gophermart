package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qutaq/gophermart/internal/domain"
	"github.com/qutaq/gophermart/internal/pgerr"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, userID int64, number string) error {
	const query = `
		INSERT INTO orders (number, user_id, status)
		VALUES ($1, $2, 'NEW')
	`

	_, err := r.db.Exec(ctx, query, number, userID)
	if err != nil {
		if pgerr.UniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("order repository: create: %w", err)
	}
	return nil
}

func (r *orderRepository) ByNumber(ctx context.Context, number string) (domain.Order, error) {
	const query = `
		SELECT number, user_id, status, accrual, uploaded_at
		FROM orders
		WHERE number = $1
	`

	var order domain.Order
	var accrual pgtype.Int8
	err := r.db.QueryRow(ctx, query, number).
		Scan(&order.Number, &order.UserID, &order.Status, &accrual, &order.UploadedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("order repository: by number: %w", err)
	}
	order.Accrual = accrualPtr(accrual)
	return order, nil
}

func (r *orderRepository) ByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	const query = `
		SELECT number, user_id, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("order repository: by user: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var order domain.Order
		var accrual pgtype.Int8
		if err := rows.Scan(
			&order.Number,
			&order.UserID,
			&order.Status,
			&accrual,
			&order.UploadedAt,
		); err != nil {
			return nil, fmt.Errorf("order repository: scan by user: %w", err)
		}
		order.Accrual = accrualPtr(accrual)
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order repository: rows by user: %w", err)
	}
	return orders, nil
}

func accrualPtr(accrual pgtype.Int8) *domain.Kopecks {
	if !accrual.Valid {
		return nil
	}
	v := domain.Kopecks(accrual.Int64)
	return &v
}
