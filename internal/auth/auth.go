package auth

import (
	"encoding/json"
	"net/http"
	"p2p-chess/internal/store"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var key jwk.Key // TODO: Load from env or file

func GenerateToken(userID string) (string, error) {
	tok, err := jwt.NewBuilder().
		Issuer("p2p-chess").
		Audience([]string{"p2p-chess"}).
		Subject(userID).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(15 * time.Minute)).
		Build()
	if err != nil {
		return "", err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, key))
	return string(signed), err
}

func ValidateToken(tokenStr string) (jwt.Token, error) {
	return jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.HS256, key), jwt.WithValidate(true))
}

type LoginRequest struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	var passwordHash string
	err = s.DB.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE handle = $1", req.Handle).Scan(&passwordHash)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	var userID string
	err = s.DB.QueryRow(r.Context(), "SELECT id FROM users WHERE handle = $1", req.Handle).Scan(&userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	token, err := GenerateToken(userID)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	_, err = s.DB.Exec(r.Context(), "INSERT INTO users (handle, password_hash, email) VALUES ($1, $2, $3)", req.Handle, string(hash), req.Email)
	if err != nil {
		http.Error(w, "Registration failed", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// TODO: Refresh token logic, loading keys from env
