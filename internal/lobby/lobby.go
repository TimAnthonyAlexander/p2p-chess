package lobby

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"p2p-chess/internal/store"

	"crypto/rand"
	"encoding/base64"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
)

func EnqueueQuickplay(s *store.Store, userID string, tc string, rated bool) error {
	queue := fmt.Sprintf("lobby:q:%s:%t", tc, rated)
	return s.Redis.RPush(context.Background(), queue, userID).Err()
}

func PairUsers(s *store.Store, tc string, rated bool) (string, string, error) {
	queue := fmt.Sprintf("lobby:q:%s:%t", tc, rated)
	users, err := s.Redis.LPopCount(context.Background(), queue, 2).Result()
	if err != nil || len(users) < 2 {
		return "", "", err
	}
	// Create match
	matchID := uuid.Must(uuid.NewV4()).String()
	_ = matchID
	// TODO: Use store to insert into matches table
	return users[0], users[1], nil
}

type QuickplayRequest struct {
	TC    string `json:"tc"`
	Rated bool   `json:"rated"`
}

func QuickplayHandler(w http.ResponseWriter, r *http.Request) {
	var req QuickplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// TODO: Get userID from auth context
	userID := "user_id_placeholder"

	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := EnqueueQuickplay(s, userID, req.TC, req.Rated); err != nil {
		http.Error(w, "Queue error", http.StatusInternalServerError)
		return
	}

	// Pairing logic (might need a separate worker, but for now synchronous)
	white, black, err := PairUsers(s, req.TC, req.Rated)
	if err != nil {
		http.Error(w, "No pair", http.StatusAccepted) // Wait or retry
		return
	}

	matchID := uuid.Must(uuid.NewV4()).String()
	// TODO: Parse tc to base_ms, inc_ms, delay_ms
	baseMs := 300000 // 5 min example
	incMs := 3000
	delayMs := 0
	msWhite := baseMs
	msBlack := baseMs

	// Insert into DB
	_, err = s.DB.Exec(r.Context(), "INSERT INTO matches (id, side_white, side_black, tc_base_ms, tc_inc_ms, tc_delay_ms, status, side_to_move, last_fen, ms_white, ms_black, rated) VALUES ($1, $2, $3, $4, $5, $6, 'live', 'w', 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1', $7, $8, $9)",
		matchID, white, black, baseMs, incMs, delayMs, msWhite, msBlack, req.Rated)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Generate matchKey (32 bytes)
	matchKey := make([]byte, 32)
	_, err = rand.Read(matchKey)
	if err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}
	matchKeyStr := base64.StdEncoding.EncodeToString(matchKey)

	// Generate joinToken (short-lived)
	joinToken := uuid.Must(uuid.NewV4()).String()

	// TODO: Mint ICE creds

	response := map[string]interface{}{
		"matchId":   matchID,
		"sides":     map[string]string{"white": white, "black": black},
		"matchKey":  matchKeyStr,
		"joinToken": joinToken,
		// "webrtcConfig": ...
	}
	json.NewEncoder(w).Encode(response)
}

func ResumeHandler(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")
	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	var lastSeq int
	var lastFen string
	var msWhite, msBlack int
	var sideToMove string
	err = s.DB.QueryRow(r.Context(), "SELECT last_seq, last_fen, ms_white, ms_black, side_to_move FROM matches WHERE id = $1", matchID).Scan(&lastSeq, &lastFen, &msWhite, &msBlack, &sideToMove)
	if err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}
	// Optional: Rotate matchKey
	newMatchKey := make([]byte, 32)
	_, err = rand.Read(newMatchKey)
	if err != nil {
		// handle
	}
	newMatchKeyStr := base64.StdEncoding.EncodeToString(newMatchKey)
	// TODO: Store new key, allow old for N events

	response := map[string]interface{}{
		"last_seq":     lastSeq,
		"fen":          lastFen,
		"msW":          msWhite,
		"msB":          msBlack,
		"side_to_move": sideToMove,
		"matchKey_new": newMatchKeyStr,
	}
	json.NewEncoder(w).Encode(response)
}

// TODO: More logic for signaling, presence
