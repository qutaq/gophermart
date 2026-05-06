package worker

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"time"

	"github.com/qutaq/gophermart/internal/accrual"
	"github.com/qutaq/gophermart/internal/domain"
)

const (
	pollInterval = 1 * time.Second
	batchSize    = 10
)

type orderRepository interface {
	Pending(ctx context.Context, limit int) iter.Seq2[domain.Order, error]
	ApplyAccrual(ctx context.Context, number, status string, accrual *domain.Kopecks, userID int64) error
}

type Poller struct {
	orders orderRepository
	client *accrual.Client
}

func New(orders orderRepository, client *accrual.Client) *Poller {
	return &Poller{orders: orders, client: client}
}

func (p *Poller) Run(ctx context.Context) {
	for {
		if err := p.poll(ctx); err != nil {
			var rateLimitErr *accrual.ErrRateLimit
			if errors.As(err, &rateLimitErr) {
				log.Printf("worker: rate limited, sleeping %s", rateLimitErr.RetryAfter)
				select {
				case <-time.After(rateLimitErr.RetryAfter):
				case <-ctx.Done():
					return
				}
				continue
			}
			log.Printf("worker: poll error: %v", err)
		}

		select {
		case <-time.After(pollInterval):
		case <-ctx.Done():
			return
		}
	}
}

func (p *Poller) poll(ctx context.Context) error {
	for order, err := range p.orders.Pending(ctx, batchSize) {
		if err != nil {
			return fmt.Errorf("worker: fetch pending: %w", err)
		}

		result, err := p.client.GetOrder(ctx, order.Number)
		if err != nil {
			var rateLimitErr *accrual.ErrRateLimit
			if errors.As(err, &rateLimitErr) {
				return err
			}
			log.Printf("worker: get order %s: %v", order.Number, err)
			continue
		}
		if result == nil {
			// 204: заказ не зарегистрирован в системе начислений
			continue
		}

		newStatus := mapStatus(result.Status)
		if err := p.orders.ApplyAccrual(ctx, order.Number, newStatus, result.Accrual, order.UserID); err != nil {
			log.Printf("worker: apply accrual for %s: %v", order.Number, err)
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
