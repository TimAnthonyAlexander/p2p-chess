package lobby

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"p2p-chess/internal/auth"
	"p2p-chess/internal/store"

	"crypto/rand"
	"encoding/base64"
	"time"

	"crypto/hmac"
	"crypto/sha1"

	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

func queueKeys(tc string, rated bool) (string, string) {
	return fmt.Sprintf("lobby:q:%s:%t", tc, rated), fmt.Sprintf("lobby:m:%s:%t", tc, rated)
}

var enqueueOnceScript = redis.NewScript(`
local q = KEYS[1]
local m = KEYS[2]
local u = ARGV[1]
if redis.call('SISMEMBER', m, u) == 1 then
  return 0
end
redis.call('SADD', m, u)
redis.call('RPUSH', q, u)
return 1
`)

func EnqueueQuickplay(s *store.Store, userID string, tc string, rated bool) (bool, error) {
	q, m := queueKeys(tc, rated)
	n, err := enqueueOnceScript.Run(context.Background(), s.Redis, []string{q, m}, userID).Int()
	return n == 1, err
}

var ErrNoPair = errors.New("no pair")

var popTwoDistinctScript = redis.NewScript(`
local q = KEYS[1]
local m = KEYS[2]
if redis.call('LLEN', q) < 2 then
  return {}
end
local a = redis.call('LPOP', q)
local b = redis.call('LPOP', q)
if a == b then
  -- put one back and wait for someone else
  redis.call('LPUSH', q, a)
  return {}
end
redis.call('SREM', m, a)
redis.call('SREM', m, b)
return {a, b}
`)

func PairUsers(s *store.Store, tc string, rated bool) (string, string, error) {
	q, m := queueKeys(tc, rated)
	res, err := popTwoDistinctScript.Run(context.Background(), s.Redis, []string{q, m}).Result()
	if err != nil {
		return "", "", err
	}

	arr, _ := res.([]interface{})
	if len(arr) < 2 {
		return "", "", ErrNoPair
	}

	w := arr[0].(string)
	b := arr[1].(string)

	// Validate both are valid UUIDs
	if _, err := uuid.FromString(w); err != nil {
		return "", "", fmt.Errorf("invalid white uuid: %w", err)
	}
	if _, err := uuid.FromString(b); err != nil {
		return "", "", fmt.Errorf("invalid black uuid: %w", err)
	}

	return w, b, nil
}

type QuickplayRequest struct {
	TC    string `json:"tc"`
	Rated bool   `json:"rated"`
}

func userIDFromAuth(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", errors.New("no bearer")
	}
	raw := strings.TrimPrefix(h, "Bearer ")
	tok, err := auth.ValidateToken(raw)
	if err != nil {
		return "", err
	}
	sub := tok.Subject()
	if sub == "" {
		return "", errors.New("no sub")
	}
	return sub, nil
}

func QuickplayHandler(w http.ResponseWriter, r *http.Request) {
	var req QuickplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TC == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	userID, err := userIDFromAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_, err = EnqueueQuickplay(s, userID, req.TC, req.Rated)
	if err != nil {
		log.Printf("enqueue error: %v", err)
		http.Error(w, "Queue error", http.StatusInternalServerError)
		return
	}

	white, black, err := PairUsers(s, req.TC, req.Rated)
	if err != nil {
		if errors.Is(err, ErrNoPair) {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"queued": true})
			return
		}
		http.Error(w, "Queue error", http.StatusInternalServerError)
		return
	}

	matchID := uuid.Must(uuid.NewV4()).String()
	baseMs, incMs, delayMs := 300000, 3000, 0
	msWhite, msBlack := baseMs, baseMs

	_, err = s.DB.Exec(r.Context(), `
INSERT INTO matches (id, side_white, side_black, tc_base_ms, tc_inc_ms, tc_delay_ms, status, side_to_move, last_fen, ms_white, ms_black, rated)
VALUES ($1,$2,$3,$4,$5,$6,'live','w','rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',$7,$8,$9)`,
		matchID, white, black, baseMs, incMs, delayMs, msWhite, msBlack, req.Rated)
	if err != nil {
		log.Printf("db insert error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	matchKey := make([]byte, 32)
	if _, err := rand.Read(matchKey); err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}
	matchKeyStr := base64.StdEncoding.EncodeToString(matchKey)
	joinToken := uuid.Must(uuid.NewV4()).String()

	turnSecret := os.Getenv("TURN_SECRET")
	iceCreds := generateTURNCreds(userID, 10*time.Minute, turnSecret)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"matchId":   matchID,
		"sides":     map[string]string{"white": white, "black": black},
		"matchKey":  matchKeyStr,
		"joinToken": joinToken,
		"webrtcConfig": map[string]any{
			"iceServers": []map[string]any{
				{"urls": "stun:your.stun.server:3478"},
				{"urls": "turn:your.turn.server:3478", "username": iceCreds["username"], "credential": iceCreds["password"]},
			},
		},
	})
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

func generateTURNCreds(userID string, ttl time.Duration, secret string) map[string]string {
	expiry := time.Now().Add(ttl).Unix()
	username := fmt.Sprintf("%d:%s", expiry, userID)
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(username))
	return map[string]string{"username": username, "password": base64.StdEncoding.EncodeToString(h.Sum(nil))}
}
