package worker

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/qutaq/gophermart/internal/accrual"
	"github.com/qutaq/gophermart/internal/domain"
)

type fakeOrderRepository struct {
	pending    []domain.Order
	pendingErr error
	applied    []appliedAccrual
	limit      int
}

type appliedAccrual struct {
	number  string
	status  string
	accrual *domain.Kopecks
	userID  int64
}

func (r *fakeOrderRepository) Pending(_ context.Context, limit int) iter.Seq2[domain.Order, error] {
	r.limit = limit
	return func(yield func(domain.Order, error) bool) {
		for _, order := range r.pending {
			if !yield(order, nil) {
				return
			}
		}
		if r.pendingErr != nil {
			yield(domain.Order{}, r.pendingErr)
		}
	}
}

func (r *fakeOrderRepository) ApplyAccrual(_ context.Context, number, status string, accrual *domain.Kopecks, userID int64) error {
	r.applied = append(r.applied, appliedAccrual{
		number:  number,
		status:  status,
		accrual: accrual,
		userID:  userID,
	})
	return nil
}

type fakeAccrualClient struct {
	results map[string]*accrual.OrderResult
	errs    map[string]error
}

func (c fakeAccrualClient) GetOrder(_ context.Context, number string) (*accrual.OrderResult, error) {
	if err := c.errs[number]; err != nil {
		return nil, err
	}
	return c.results[number], nil
}

func TestPollAppliesAccrualFromClient(t *testing.T) {
	sum := domain.Kopecks(12345)
	orders := &fakeOrderRepository{
		pending: []domain.Order{
			{Number: "12345678903", UserID: 7},
		},
	}
	client := fakeAccrualClient{
		results: map[string]*accrual.OrderResult{
			"12345678903": {Order: "12345678903", Status: "PROCESSED", Accrual: &sum},
		},
	}

	poller := New(orders, client)

	if err := poller.poll(context.Background()); err != nil {
		t.Fatalf("poll returned error: %v", err)
	}
	if len(orders.applied) != 1 {
		t.Fatalf("expected 1 applied accrual, got %d", len(orders.applied))
	}

	applied := orders.applied[0]
	if applied.number != "12345678903" {
		t.Errorf("number = %q, want %q", applied.number, "12345678903")
	}
	if applied.status != "PROCESSED" {
		t.Errorf("status = %q, want %q", applied.status, "PROCESSED")
	}
	if applied.accrual != &sum {
		t.Errorf("accrual pointer = %p, want %p", applied.accrual, &sum)
	}
	if applied.userID != 7 {
		t.Errorf("userID = %d, want %d", applied.userID, 7)
	}
}

func TestPollReturnsRateLimitError(t *testing.T) {
	rateLimitErr := &accrual.ErrRateLimit{RetryAfter: time.Second}
	orders := &fakeOrderRepository{
		pending: []domain.Order{
			{Number: "12345678903", UserID: 7},
		},
	}
	client := fakeAccrualClient{
		errs: map[string]error{"12345678903": rateLimitErr},
	}

	poller := New(orders, client)

	err := poller.poll(context.Background())
	if !errors.Is(err, rateLimitErr) {
		t.Fatalf("poll error = %v, want %v", err, rateLimitErr)
	}
	if len(orders.applied) != 0 {
		t.Fatalf("expected no applied accruals, got %d", len(orders.applied))
	}
}

func TestPollUsesConfiguredBatchSize(t *testing.T) {
	orders := &fakeOrderRepository{}
	client := fakeAccrualClient{}
	poller := New(orders, client, WithBatchSize(3))

	if err := poller.poll(context.Background()); err != nil {
		t.Fatalf("poll returned error: %v", err)
	}
	if orders.limit != 3 {
		t.Fatalf("batch size = %d, want 3", orders.limit)
	}
}
