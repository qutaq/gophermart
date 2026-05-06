package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/qutaq/gophermart/internal/auth"
	"github.com/qutaq/gophermart/internal/domain"
	"github.com/qutaq/gophermart/internal/repository"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	credentials, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	passwordHash, err := auth.HashPassword(credentials.Password)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	user, err := h.users.Create(r.Context(), credentials.Login, passwordHash)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.authenticate(w, user.ID)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	credentials, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.users.ByLogin(r.Context(), credentials.Login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !auth.CheckPassword(credentials.Password, user.PasswordHash) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	h.authenticate(w, user.ID)
}

func (h *Handler) authenticate(w http.ResponseWriter, userID int64) {
	token, err := auth.NewToken(userID, h.jwtSecret)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	auth.SetAuthCookie(w, token)
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (domain.Credentials, bool) {
	var credentials domain.Credentials
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credentials); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return domain.Credentials{}, false
	}
	if credentials.Login == "" || credentials.Password == "" ||
		strings.TrimSpace(credentials.Login) != credentials.Login {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return domain.Credentials{}, false
	}
	return credentials, true
}
