package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/qutaq/gophermart/internal/domain"
)

type OrderResult struct {
	Order   string          `json:"order"`
	Status  string          `json:"status"`
	Accrual *domain.Kopecks `json:"accrual"`
}

type ErrRateLimit struct {
	RetryAfter time.Duration
}

func (e *ErrRateLimit) Error() string {
	return fmt.Sprintf("accrual: rate limit, retry after %s", e.RetryAfter)
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetOrder(ctx context.Context, number string) (*OrderResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/orders/"+number, nil)
	if err != nil {
		return nil, fmt.Errorf("accrual: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("accrual: do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result OrderResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("accrual: decode response: %w", err)
		}
		return &result, nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusTooManyRequests:
		return nil, &ErrRateLimit{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	default:
		return nil, fmt.Errorf("accrual: unexpected status %d", resp.StatusCode)
	}
}

func parseRetryAfter(header string) time.Duration {
	if n, err := strconv.Atoi(header); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 60 * time.Second
}
