package referee

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"

	"p2p-chess/internal/clock"
	"p2p-chess/internal/store"

	chess "github.com/corentings/chess/v2"
)

type ValidationResult struct {
	NewFEN  string
	Outcome chess.Outcome
	Method  chess.Method
	Valid   bool
}

// Extended ValidateMove to return result struct
func ValidateMoveWithOutcome(fenStr string, uciMove string) (*ValidationResult, error) {
	fenOpt, err := chess.FEN(fenStr)
	if err != nil {
		return &ValidationResult{Valid: false}, err
	}
	g := chess.NewGame(fenOpt)
	for _, mv := range g.ValidMoves() {
		if mv.String() == uciMove {
			if err := g.Move(&mv, nil); err != nil {
				return &ValidationResult{Valid: false}, err
			}
			newFEN := g.Position().String()
			return &ValidationResult{
				NewFEN:  newFEN,
				Outcome: g.Outcome(),
				Method:  g.Method(),
				Valid:   true,
			}, nil
		}
	}
	return &ValidationResult{Valid: false}, fmt.Errorf("illegal move: %s", uciMove)
}

func ComputeZobrist(fen string) string {
	hash := sha256.Sum256([]byte(fen))
	return hex.EncodeToString(hash[:])
}

// TODO: If library has built-in Zobrist, use that instead

// TODO: Full adjudication, clocks, outcomes

type AppendRequest struct {
	Seq      int    `json:"seq"`
	UCI      string `json:"uci"`
	FEN      string `json:"fen"`
	MsWhite  int    `json:"msWhite"`
	MsBlack  int    `json:"msBlack"`
	TsClient string `json:"tsClient"`
	Side     string `json:"side"`
	Sig      string `json:"sig"`
}

func AppendHandler(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")
	var req AppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// TODO: Fetch matchKey from Redis or DB
	matchKey := []byte("placeholder_key")

	// Verify sig = HMAC-SHA256(matchKey, canonical(seq|uci|fen|msW|msB))
	canonical := fmt.Sprintf("%d|%s|%s|%d|%d", req.Seq, req.UCI, req.FEN, req.MsWhite, req.MsBlack)
	computedSig := hmac.New(sha256.New, matchKey)
	computedSig.Write([]byte(canonical))
	expectedSig := hex.EncodeToString(computedSig.Sum(nil))
	if req.Sig != expectedSig {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Validate move
	result, err := ValidateMoveWithOutcome(req.FEN, req.UCI)
	if err != nil || !result.Valid {
		// TODO: Emit correction via WS
		http.Error(w, "Invalid move", http.StatusBadRequest)
		return
	}

	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tsClientTime, err := time.Parse(time.RFC3339, req.TsClient)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}
	tsServer := time.Now()

	// TODO: Fetch current clock state from DB or Redis
	clockState := &clock.ClockState{
		MsWhite:    req.MsWhite,
		MsBlack:    req.MsBlack,
		SideToMove: req.Side, // assuming
		// LastTicks, Inc, Delay from match
	}

	newWhite, newBlack, err := clock.UpdateClocks(clockState, req.Side, tsServer, tsClientTime)
	if err != nil {
		reason := "timeout"
		resultStr := ""
		if req.Side == "w" {
			resultStr = "0-1"
		} else {
			resultStr = "1-0"
		}
		_, _ = s.DB.Exec(r.Context(), "UPDATE matches SET status = 'finished', result = $1, reason = $2, finished_at = NOW() WHERE id = $3", resultStr, reason, matchID)
		// Notify
		http.Error(w, "Timeout", http.StatusBadRequest)
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"uci": req.UCI, "fen_before": req.FEN, "fen_after": result.NewFEN, "msW": newWhite, "msB": newBlack})

	zobrist := ComputeZobrist(result.NewFEN)

	// Insert and update with zobrist used
	_, err = s.DB.Exec(r.Context(), "INSERT INTO match_events (match_id, seq, type, payload, side, ts_client, ts_server, zobrist, sig, valid) VALUES ($1, $2, 'move', $3, $4, $5, NOW(), $6, $7, true)",
		matchID, req.Seq, payload, req.Side, req.TsClient, zobrist, req.Sig)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Update matches with new clocks
	_, err = s.DB.Exec(r.Context(), "UPDATE matches SET last_seq = $1, last_fen = $2, ms_white = $3, ms_black = $4, side_to_move = CASE WHEN side_to_move = 'w' THEN 'b' ELSE 'w' END WHERE id = $5",
		req.Seq, result.NewFEN, newWhite, newBlack, matchID)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	// Check for terminal state
	if result.Outcome != chess.NoOutcome {
		// Update status, result, reason
		reason := string(result.Method)
		resultStr := string(result.Outcome)
		_, err = s.DB.Exec(r.Context(), "UPDATE matches SET status = 'finished', result = $1, reason = $2, finished_at = NOW() WHERE id = $3", resultStr, reason, matchID)
		if err != nil {
			// log error
		}
		// TODO: Notify via WS
		if err == nil {
			s.UpdateRatings(matchID)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func SpectateHandler(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	s, err := store.New()
	if err != nil {
		return
	}

	rows, err := s.DB.Query(r.Context(), "SELECT seq, type, payload FROM match_events WHERE match_id = $1 ORDER BY seq", matchID)
	if err != nil {
		return
	}
	defer rows.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	for rows.Next() {
		var seq int
		var typ string
		var payload json.RawMessage
		if err := rows.Scan(&seq, &typ, &payload); err != nil {
			break
		}
		fmt.Fprintf(w, "data: %s\n\n", payload)
		flusher.Flush()
	}

	// TODO: Tail for live updates using pub/sub or polling
}
