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

func EnqueueQuickplay(s *store.Store, userID string, tc string, rated bool) error {
	queue := fmt.Sprintf("lobby:q:%s:%t", tc, rated)
	return s.Redis.RPush(context.Background(), queue, userID).Err()
}

var ErrNoPair = errors.New("no pair")

var popTwoScript = redis.NewScript(`
local k = KEYS[1]
if redis.call('LLEN', k) >= 2 then
  local a = redis.call('LPOP', k)
  local b = redis.call('LPOP', k)
  return {a, b}
else
  return {}
end`)

func PairUsers(s *store.Store, tc string, rated bool) (string, string, error) {
	queue := fmt.Sprintf("lobby:q:%s:%t", tc, rated)
	res, err := popTwoScript.Run(context.Background(), s.Redis, []string{queue}).Result()
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
	log.Printf("Authorization header: %s", h)

	if !strings.HasPrefix(h, "Bearer ") {
		log.Printf("Auth error: missing Bearer prefix")
		return "", errors.New("no bearer")
	}
	raw := strings.TrimPrefix(h, "Bearer ")

	// Log the first few characters of the token for debugging
	if len(raw) > 10 {
		log.Printf("Token prefix: %s...", raw[:10])
	}

	tok, err := auth.ValidateToken(raw)
	if err != nil {
		log.Printf("Token validation error: %T %v", err, err)
		return "", err
	}

	sub := tok.Subject()
	log.Printf("Token subject: %q", sub)

	if sub == "" {
		log.Printf("Auth error: empty subject")
		return "", errors.New("no sub")
	}
	return sub, nil
}

func QuickplayHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("QuickplayHandler called with method: %s", r.Method)

	var req QuickplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Request decode error: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Request data: tc=%s rated=%t", req.TC, req.Rated)

	// Validate TC format upfront (optional but helps catch errors early)
	if req.TC == "" {
		http.Error(w, "Invalid time control", http.StatusBadRequest)
		return
	}

	// Check if auth package is initialized
	token := r.Header.Get("Authorization")
	log.Printf("Raw Authorization header: %q", token)

	uid, err := userIDFromAuth(r)
	if err != nil {
		log.Printf("Authentication error: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("Authenticated user ID: %s", uid)
	userID := uid

	s, err := store.New()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := EnqueueQuickplay(s, userID, req.TC, req.Rated); err != nil {
		log.Printf("enqueue error: %v", err)
		http.Error(w, "Queue error", http.StatusInternalServerError)
		return
	}

	// Pairing logic (might need a separate worker, but for now synchronous)
	white, black, err := PairUsers(s, req.TC, req.Rated)
	if err != nil {
		if errors.Is(err, ErrNoPair) {
			http.Error(w, "No pair", http.StatusAccepted) // Wait or retry
			return
		}
		log.Printf("pairing error: %v", err)
		http.Error(w, "Queue error", http.StatusInternalServerError)
		return
	}

	matchID := uuid.Must(uuid.NewV4()).String()
	// TODO: Parse tc to base_ms, inc_ms, delay_ms
	baseMs := 300000 // 5 min example
	incMs := 3000
	delayMs := 0
	msWhite := baseMs
	msBlack := baseMs

	// Extra validation for player UUIDs
	if _, err := uuid.FromString(white); err != nil {
		http.Error(w, "Invalid white player ID", http.StatusInternalServerError)
		return
	}
	if _, err := uuid.FromString(black); err != nil {
		http.Error(w, "Invalid black player ID", http.StatusInternalServerError)
		return
	}

	// Log for debugging
	log.Printf("pair: white=%q black=%q tc=%s rated=%t", white, black, req.TC, req.Rated)

	// Insert into DB
	_, err = s.DB.Exec(r.Context(), "INSERT INTO matches (id, side_white, side_black, tc_base_ms, tc_inc_ms, tc_delay_ms, status, side_to_move, last_fen, ms_white, ms_black, rated) VALUES ($1, $2, $3, $4, $5, $6, 'live', 'w', 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1', $7, $8, $9)",
		matchID, white, black, baseMs, incMs, delayMs, msWhite, msBlack, req.Rated)
	if err != nil {
		log.Printf("db insert error: %T %v", err, err)
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
	turnSecret := os.Getenv("TURN_SECRET")
	iceCreds := generateTURNCreds(userID, 10*time.Minute, turnSecret)
	response["webrtcConfig"] = map[string]interface{}{
		"iceServers": []map[string]interface{}{
			{"urls": "stun:your.stun.server:3478"},
			{"urls": "turn:your.turn.server:3478", "username": iceCreds["username"], "credential": iceCreds["password"]},
		},
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

func generateTURNCreds(userID string, ttl time.Duration, secret string) map[string]string {
	expiry := time.Now().Add(ttl).Unix()
	username := fmt.Sprintf("%d:%s", expiry, userID)
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(username))
	password := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return map[string]string{
		"username": username,
		"password": password,
	}
}
