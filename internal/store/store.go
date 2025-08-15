package store

import (
	"context"
	"os"

	"github.com/SheerLuck/glicko2" // Public library
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func New() (*Store, error) {
	db, err := pgxpool.New(context.Background(), os.Getenv("DB_DSN"))
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	return &Store{DB: db, Redis: rdb}, nil
}

func (s *Store) UpdateRatings(matchID string) error {
	// Fetch match result, players, rated
	var white, black string
	var result string
	var rated bool
	err := s.DB.QueryRow(context.Background(), "SELECT side_white, side_black, result, rated FROM matches WHERE id = $1", matchID).Scan(&white, &black, &result, &rated)
	if err != nil || !rated || result == "" {
		return nil
	}

	// Fetch current ratings
	var whiteRating, whiteRD, whiteVol, blackRating, blackRD, blackVol float64
	err = s.DB.QueryRow(context.Background(), "SELECT rating, rd, volatility FROM ratings WHERE user_id = $1", white).Scan(&whiteRating, &whiteRD, &whiteVol)
	if err != nil {
		// Default if not found
		whiteRating = 1500
		whiteRD = 350
		whiteVol = 0.06
	}
	// Similar for black

	// Create players
	whitePlayer := glicko2.NewPlayer(whiteRating, whiteRD, whiteVol)
	blackPlayer := glicko2.NewPlayer(blackRating, blackRD, blackVol)

	// Determine score
	var whiteScore float64
	switch result {
	case "1-0":
		whiteScore = 1.0
	case "0-1":
		whiteScore = 0.0
	case "1/2-1/2":
		whiteScore = 0.5
	default:
		return nil
	}

	// Update
	whitePlayer.Update(blackPlayer, whiteScore)
	blackPlayer.Update(whitePlayer, 1-whiteScore)

	// Save back to DB
	_, err = s.DB.Exec(context.Background(), "INSERT INTO ratings (user_id, rating, rd, volatility, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (user_id) DO UPDATE SET rating = $2, rd = $3, volatility = $4, updated_at = NOW()", white, whitePlayer.Rating, whitePlayer.RD, whitePlayer.Volatility)
	// Similar for black

	return err
}

// Call this after setting match to finished if rated
