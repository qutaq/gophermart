package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/qutaq/gophermart/internal/auth"
	"github.com/qutaq/gophermart/internal/repository"
)

type Handler struct {
	users     repository.UserRepository
	orders    repository.OrderRepository
	jwtSecret []byte
}

func New(users repository.UserRepository, orders repository.OrderRepository, jwtSecret []byte) *Handler {
	return &Handler{
		users:     users,
		orders:    orders,
		jwtSecret: jwtSecret,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(h.jwtSecret))
		r.Post("/api/user/orders", h.UploadOrder)
		r.Get("/api/user/orders", h.ListOrders)
	})
}
