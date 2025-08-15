package admin

import (
	"net/http"
	"p2p-chess/internal/store"

	"github.com/go-chi/chi"
)

func AdminBanHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	s, _ := store.New()
	_, err := s.DB.Exec(r.Context(), "UPDATE users SET banned = true WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func AdminAbortHandler(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "matchID")
	s, _ := store.New()
	_, err := s.DB.Exec(r.Context(), "UPDATE matches SET status = 'aborted', reason = 'admin' WHERE id = $1", matchID)
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	// TODO: Notify players via WS
	w.WriteHeader(http.StatusOK)
}

// More admin functions
