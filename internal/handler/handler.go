package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/qutaq/gophermart/internal/auth"
	"github.com/qutaq/gophermart/internal/middleware"
	"github.com/qutaq/gophermart/internal/repository"
)

type Handler struct {
	users       repository.UserRepository
	orders      repository.OrderRepository
	withdrawals repository.WithdrawalRepository
	jwtSecret   []byte
}

func New(
	users repository.UserRepository,
	orders repository.OrderRepository,
	withdrawals repository.WithdrawalRepository,
	jwtSecret []byte,
) *Handler {
	return &Handler{
		users:       users,
		orders:      orders,
		withdrawals: withdrawals,
		jwtSecret:   jwtSecret,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)

	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(h.jwtSecret))
		r.Post("/api/user/orders", h.UploadOrder)
		r.Get("/api/user/orders", h.ListOrders)
		r.Get("/api/user/balance", h.Balance)
		r.Post("/api/user/balance/withdraw", h.Withdraw)
		r.Get("/api/user/withdrawals", h.Withdrawals)
	})
}
