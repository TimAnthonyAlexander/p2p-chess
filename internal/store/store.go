package store

import (
	"context"
	"math"
	"os"

	glicko "github.com/gregandcin/go-glicko2"
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

// Simple Elo placeholder
const K = 32.0 // K-factor

func (s *Store) UpdateElo(matchID string) error {
	var white, black string
	var result string
	var rated bool
	err := s.DB.QueryRow(context.Background(), "SELECT side_white, side_black, result, rated FROM matches WHERE id = $1", matchID).Scan(&white, &black, &result, &rated)
	if err != nil || !rated || result == "" {
		return nil
	}

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

	var whiteR, blackR float64
	err = s.DB.QueryRow(context.Background(), "SELECT rating FROM ratings WHERE user_id = $1", white).Scan(&whiteR)
	if err != nil {
		whiteR = 1500
	}
	err = s.DB.QueryRow(context.Background(), "SELECT rating FROM ratings WHERE user_id = $1", black).Scan(&blackR)
	if err != nil {
		blackR = 1500
	}

	expectedWhite := 1.0 / (1.0 + math.Pow(10, (blackR-whiteR)/400))
	delta := K * (whiteScore - expectedWhite)

	newWhite := whiteR + delta
	newBlack := blackR - delta

	_, err = s.DB.Exec(context.Background(), "INSERT INTO ratings (user_id, rating, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET rating = $2, updated_at = NOW()", white, newWhite)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(context.Background(), "INSERT INTO ratings (user_id, rating, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET rating = $2, updated_at = NOW()", black, newBlack)
	return err
}

func (s *Store) UpdateRatings(matchID string) error {
	return s.UpdateElo(matchID)
}

// Call this after setting match to finished if rated

func (s *Store) GetLeaderboard(limit int) ([]map[string]interface{}, error) {
	rows, err := s.DB.Query(context.Background(), "SELECT u.handle, r.rating FROM ratings r JOIN users u ON r.user_id = u.id ORDER BY r.rating DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []map[string]interface{}
	for rows.Next() {
		var handle string
		var rating float64
		if err := rows.Scan(&handle, &rating); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, map[string]interface{}{"handle": handle, "rating": rating})
	}
	return leaderboard, nil
}

type Rating struct {
	R, RD, Sigma float64
}

func (s *Store) UpdateGlickoPeriod(p1, p2 Rating, score1 float64) (Rating, Rating) {
	a := glicko.NewPlayer(glicko.NewRating(p1.R, p1.RD, p1.Sigma))
	b := glicko.NewPlayer(glicko.NewRating(p2.R, p2.RD, p2.Sigma))
	period := glicko.NewRatingPeriod()
	var res glicko.MatchResult
	switch score1 {
	case 1:
		res = glicko.MATCH_RESULT_WIN
	case 0.5:
		res = glicko.MATCH_RESULT_DRAW
	default:
		res = glicko.MATCH_RESULT_LOSS
	}
	period.AddMatch(a, b, res)
	period.Calculate()
	ra := a.Rating()
	rb := b.Rating()
	return Rating{ra.R(), ra.Rd(), ra.Sigma()}, Rating{rb.R(), rb.Rd(), rb.Sigma()}
}
