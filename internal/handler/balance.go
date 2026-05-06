package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/qutaq/gophermart/internal/auth"
	"github.com/qutaq/gophermart/internal/domain"
	"github.com/qutaq/gophermart/internal/luhn"
	"github.com/qutaq/gophermart/internal/repository"
)

type balanceResponse struct {
	Current   domain.Kopecks `json:"current"`
	Withdrawn domain.Kopecks `json:"withdrawn"`
}

type withdrawRequest struct {
	Order string         `json:"order"`
	Sum   domain.Kopecks `json:"sum"`
}

type withdrawalResponse struct {
	Order       string         `json:"order"`
	Sum         domain.Kopecks `json:"sum"`
	ProcessedAt string         `json:"processed_at"`
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	current, withdrawn, err := h.users.Balance(r.Context(), userID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balanceResponse{Current: current, Withdrawn: withdrawn})
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !luhn.Valid(req.Order) {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	err := h.users.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

}

func (h *Handler) Withdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	list, err := h.withdrawals.ByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]withdrawalResponse, 0, len(list))
	for _, wd := range list {
		response = append(response, withdrawalResponse{
			Order:       wd.OrderNumber,
			Sum:         wd.Sum,
			ProcessedAt: wd.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
