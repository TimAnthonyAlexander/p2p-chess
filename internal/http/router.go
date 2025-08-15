package http

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"p2p-chess/internal/admin"
	"p2p-chess/internal/auth"
	"p2p-chess/internal/lobby"
	"p2p-chess/internal/referee"
	"p2p-chess/internal/store"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		return o == "http://localhost:5174" || o == "http://localhost:5174/"
	},
}

func writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "http://localhost:5174" || origin == "http://localhost:5174/" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
			w.Header().Set("Access-Control-Allow-Headers", req)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "300")
	}
}

func preflight(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w, r)
		if r.Method == http.MethodOptions {
			preflight(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	// CORS first
	r.Use(CorsMiddleware)

	// Ensure OPTIONS never 405s on this router
	r.MethodFunc(http.MethodOptions, "/*", preflight)

	// Safety: if method doesn’t match but it’s OPTIONS, still return 204
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodOptions {
			preflight(w, req)
			return
		}
		// include CORS even on 405 so browsers still see headers
		writeCORS(w, req)
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Auth
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(10, 1))
		r.Post("/v1/auth/login", auth.LoginHandler)
		r.Post("/v1/auth/register", auth.RegisterHandler)
	})

	// Matchmaking
	r.Group(func(r chi.Router) {
		r.Use(RateLimitMiddleware(5, 1))
		r.Post("/v1/match/quick", lobby.QuickplayHandler)
	})

	// Append/Resume
	r.Post("/v1/match/{id}/append", referee.AppendHandler)
	r.Post("/v1/match/{id}/resume", lobby.ResumeHandler)

	// WS signaling
	r.Get("/v1/ws/signal", SignalingWS)

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

	// Admin
	r.Group(func(r chi.Router) {
		r.Use(AdminMiddleware)
		r.Post("/v1/admin/ban/{userID}", admin.AdminBanHandler)
		r.Post("/v1/admin/abort/{matchID}", admin.AdminAbortHandler)
	})

	return r
}

func SignalingWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func isAdmin(r *http.Request) bool {
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

func RateLimitMiddleware(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	type key struct{ ip string }
	limiters := sync.Map{}

	get := func(ip string) *rate.Limiter {
		k := key{ip}
		if v, ok := limiters.Load(k); ok {
			return v.(*rate.Limiter)
		}
		l := rate.NewLimiter(rps, burst)
		actual, _ := limiters.LoadOrStore(k, l)
		return actual.(*rate.Limiter)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}
			if !get(ip).Allow() {
				http.Error(w, "Rate limited", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
