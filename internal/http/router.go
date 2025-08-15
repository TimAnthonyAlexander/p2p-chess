package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"p2p-chess/internal/auth"
	"p2p-chess/internal/lobby"
	"p2p-chess/internal/referee"
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
	r.Post("/v1/auth/login", auth.LoginHandler) // Assuming handler in auth
	r.Post("/v1/auth/register", auth.RegisterHandler)
	// Matchmaking
	r.Post("/v1/match/quick", lobby.QuickplayHandler) // Assuming handler
	// Append
	r.Post("/v1/match/{id}/append", referee.AppendHandler) // Assuming
	// Resume
	r.Post("/v1/match/{id}/resume", lobby.ResumeHandler) // Assuming in lobby
	// WS signaling
	r.Get("/v1/ws/signal", SignalingWS) // To implement
	// Spectator SSE
	r.Get("/v1/match/{id}/spectate", referee.SpectateHandler)
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
