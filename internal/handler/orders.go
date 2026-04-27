package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/qutaq/gophermart/internal/auth"
	"github.com/qutaq/gophermart/internal/domain"
	"github.com/qutaq/gophermart/internal/luhn"
	"github.com/qutaq/gophermart/internal/repository"
)

type orderResponse struct {
	Number     string          `json:"number"`
	Status     string          `json:"status"`
	Accrual    *domain.Kopecks `json:"accrual,omitempty"`
	UploadedAt string          `json:"uploaded_at"`
}

func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if !isTextPlain(r.Header.Get("Content-Type")) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(string(body))
	if !luhn.Valid(number) {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	order, err := h.orders.ByNumber(r.Context(), number)
	if err == nil {
		writeKnownOrderStatus(w, order, userID)
		return
	}
	if !errors.Is(err, repository.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := h.orders.Create(r.Context(), userID, number); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.writeConflictingOrderStatus(w, r, number, userID)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orders, err := h.orders.ByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, newOrderResponse(order))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) writeConflictingOrderStatus(w http.ResponseWriter, r *http.Request, number string, userID int64) {
	order, err := h.orders.ByNumber(r.Context(), number)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writeKnownOrderStatus(w, order, userID)
}

func writeKnownOrderStatus(w http.ResponseWriter, order domain.Order, userID int64) {
	if order.UserID == userID {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
}

func newOrderResponse(order domain.Order) orderResponse {
	return orderResponse{
		Number:     order.Number,
		Status:     order.Status,
		Accrual:    order.Accrual,
		UploadedAt: order.UploadedAt.Format(time.RFC3339),
	}
}

func isTextPlain(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "text/plain"
}
