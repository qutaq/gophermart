package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testSecret = []byte("test-secret")

func TestHashAndCheckPassword(t *testing.T) {
	password := "super-secret-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == password {
		t.Fatal("hash must differ from plaintext")
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword returned false for correct password")
	}
	if CheckPassword("wrong", hash) {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestNewTokenAndParseToken(t *testing.T) {
	const userID int64 = 42

	token, err := NewToken(userID, testSecret)
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}

	got, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if got != userID {
		t.Errorf("ParseToken userID = %d, want %d", got, userID)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, _ := NewToken(1, testSecret)
	_, err := ParseToken(token, []byte("other-secret"))
	if err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := ParseToken("not.a.token", testSecret)
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestRequireAuth_NoToken(t *testing.T) {
	handler := RequireAuth(testSecret)(okHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuth_ValidHeader(t *testing.T) {
	token, _ := NewToken(7, testSecret)

	handler := RequireAuth(testSecret)(captureIDHandler(t, 7))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(AuthHeaderName, "Bearer "+token)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireAuth_ValidCookie(t *testing.T) {
	token, _ := NewToken(99, testSecret)

	handler := RequireAuth(testSecret)(captureIDHandler(t, 99))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: token})
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}

func captureIDHandler(t *testing.T, wantID int64) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Error("userID not found in context")
		}
		if id != wantID {
			t.Errorf("userID = %d, want %d", id, wantID)
		}
	})
}
