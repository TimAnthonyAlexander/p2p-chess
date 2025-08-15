package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"p2p-chess/internal/store"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var hmacKey []byte

func Init() error {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		return errors.New("JWT_SECRET not set")
	}
	hmacKey = []byte(sec)
	return nil
}

func GenerateToken(userID, role string) (string, error) {
	if len(hmacKey) == 0 {
		return "", errors.New("auth not initialized: empty key")
	}
	now := time.Now().UTC()
	tok, err := jwt.NewBuilder().
		Issuer("p2p-chess").
		Audience([]string{"p2p-chess"}).
		Subject(userID).
		IssuedAt(now).
		Expiration(now.Add(15*time.Minute)).
		Claim("role", role).
		Build()
	if err != nil {
		return "", err
	}
	b, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, hmacKey))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ValidateToken(tokenStr string) (jwt.Token, error) {
	if len(hmacKey) == 0 {
		return nil, errors.New("auth not initialized: empty key")
	}
	// Optional: strip "Bearer "
	if strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
		tokenStr = tokenStr[7:]
	}
	return jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.HS256, hmacKey), jwt.WithValidate(true))
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
	// Set content type header first thing
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	s, err := store.New()
	if err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var passwordHash string
	err = s.DB.QueryRow(r.Context(), "SELECT password_hash FROM users WHERE handle = $1", req.Handle).Scan(&passwordHash)
	if err != nil {
		jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	var userID string
	err = s.DB.QueryRow(r.Context(), "SELECT id FROM users WHERE handle = $1", req.Handle).Scan(&userID)
	if err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	token, err := GenerateToken(userID, "user") // Default role for now
	if err != nil {
		log.Printf("GenerateToken error: %v", err)
		jsonError(w, "Error generating authentication token", http.StatusInternalServerError)
		return
	}

	// Set the content type header explicitly
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type header first thing
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	s, err := store.New()
	if err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	_, err = s.DB.Exec(r.Context(), "INSERT INTO users (handle, password_hash, email) VALUES ($1, $2, $3)", req.Handle, string(hash), req.Email)
	if err != nil {
		jsonError(w, "Registration failed - username or email may already be taken", http.StatusConflict)
		return
	}

	// Set the content type header explicitly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// Helper function for returning JSON errors
func jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// TODO: Refresh token logic, loading keys from env
