package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"p2p-chess/internal/admin"
	"p2p-chess/internal/auth"
	"p2p-chess/internal/lobby"
	"p2p-chess/internal/referee"
	"p2p-chess/internal/store"

	"golang.org/x/time/rate"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	// Auth
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(10, 1)) // e.g., 10 rps, burst 1 for auth
		r.Post("/v1/auth/login", auth.LoginHandler)
		r.Post("/v1/auth/register", auth.RegisterHandler)
	})
	// Matchmaking
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(5, 1)) // for matchmaking
		r.Post("/v1/match/quick", lobby.QuickplayHandler)
	})
	// Append
	r.Post("/v1/match/{id}/append", referee.AppendHandler) // Assuming
	// Resume
	r.Post("/v1/match/{id}/resume", lobby.ResumeHandler) // Assuming in lobby
	// WS signaling
	r.Get("/v1/ws/signal", SignalingWS) // To implement
	// Spectator SSE
	r.Get("/v1/match/{id}/spectate", referee.SpectateHandler)
	// Leaderboard
	r.Get("/v1/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		s, _ := store.New()
		leaderboard, err := s.GetLeaderboard(10)
		if err != nil {
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(leaderboard)
	})
	// Admin endpoints
	r.Group(func(r chi.Router) {
		r.Use(AdminMiddleware)
		r.Post("/v1/admin/ban/{userID}", admin.AdminBanHandler)
		r.Post("/v1/admin/abort/{matchID}", admin.AdminAbortHandler)
		// More
	})
	// TODO: More endpoints like resume, spectate
	return r
}

func SignalingWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// log error
		return
	}
	defer conn.Close()

	for {
		// Read message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// TODO: Parse JSON, handle types like join, offer, ice, based on proto
		// Relay to peers, use channels or Redis pub/sub for multi-user
		// Example: conn.WriteMessage(websocket.TextMessage, msg)
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func isAdmin(r *http.Request) bool {
	// Parse JWT from header, check role == "admin"
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		return false
	}
	token, err := auth.ValidateToken(tokenStr)
	if err != nil {
		return false
	}
	role, _ := token.Get("role")
	return role == "admin"
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddleware(limit rate.Limit, burst int) chi.Middleware {
	limiter := rate.NewLimiter(limit, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Rate limited", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Define handlers in a new internal/admin package or here
