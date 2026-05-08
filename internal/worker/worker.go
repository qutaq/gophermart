package worker

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"time"

	"github.com/qutaq/gophermart/internal/accrual"
	"github.com/qutaq/gophermart/internal/domain"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultBatchSize    = 10
)

type Option[T any] func(*T)

type orderRepository interface {
	Pending(ctx context.Context, limit int) iter.Seq2[domain.Order, error]
	ApplyAccrual(ctx context.Context, number, status string, accrual *domain.Kopecks, userID int64) error
}

type accrualClient interface {
	GetOrder(ctx context.Context, number string) (*accrual.OrderResult, error)
}

type Poller struct {
	orders       orderRepository
	client       accrualClient
	pollInterval time.Duration
	batchSize    int
}

func New(orders orderRepository, client accrualClient, opts ...Option[Poller]) *Poller {
	poller := &Poller{
		orders:       orders,
		client:       client,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
	}
	for _, opt := range opts {
		opt(poller)
	}
	return poller
}

func WithPollInterval(interval time.Duration) Option[Poller] {
	return func(p *Poller) {
		if interval > 0 {
			p.pollInterval = interval
		}
	}
}

func WithBatchSize(size int) Option[Poller] {
	return func(p *Poller) {
		if size > 0 {
			p.batchSize = size
		}
	}
}

func (p *Poller) Run(ctx context.Context) error {
	for {
		if err := p.poll(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			var rateLimitErr *accrual.ErrRateLimit
			if errors.As(err, &rateLimitErr) {
				slog.Info("worker: rate limited, sleeping", "retry_after", rateLimitErr.RetryAfter)
				select {
				case <-time.After(rateLimitErr.RetryAfter):
				case <-ctx.Done():
					return nil
				}
				continue
			}
			slog.Error("worker: poll error", "error", err)
		}

		select {
		case <-time.After(p.pollInterval):
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Poller) poll(ctx context.Context) error {
	for order, err := range p.orders.Pending(ctx, p.batchSize) {
		if err != nil {
			return fmt.Errorf("worker: fetch pending: %w", err)
		}

		result, err := p.client.GetOrder(ctx, order.Number)
		if err != nil {
			var rateLimitErr *accrual.ErrRateLimit
			if errors.As(err, &rateLimitErr) {
				return err
			}
			slog.Error("worker: get order", "number", order.Number, "error", err)
			continue
		}
		if result == nil {
			// 204: заказ не зарегистрирован в системе начислений
			continue
		}

		newStatus := mapStatus(result.Status)
		if err := p.orders.ApplyAccrual(ctx, order.Number, newStatus, result.Accrual, order.UserID); err != nil {
			slog.Error("worker: apply accrual", "number", order.Number, "error", err)
		}
	}

	return nil
}

func mapStatus(accrualStatus string) string {
	switch accrualStatus {
	case "INVALID":
		return "INVALID"
	case "PROCESSED":
		return "PROCESSED"
	default:
		return "PROCESSING"
	}
}
