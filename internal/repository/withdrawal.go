package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qutaq/gophermart/internal/domain"
)

type withdrawalRepository struct {
	db *pgxpool.Pool
}

func NewWithdrawalRepository(db *pgxpool.Pool) WithdrawalRepository {
	return &withdrawalRepository{db: db}
}

func (r *withdrawalRepository) ByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	const query = `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("withdrawal repository: by user: %w", err)
	}
	defer rows.Close()

	withdrawals := make([]domain.Withdrawal, 0)
	for rows.Next() {
		var w domain.Withdrawal
		if err := rows.Scan(&w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("withdrawal repository: scan: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("withdrawal repository: rows: %w", err)
	}
	return withdrawals, nil
}
