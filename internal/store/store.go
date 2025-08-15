package store

import (
	"context"
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

type RatingRow struct{ R, RD, Sigma float64 }

func (s *Store) UpdateInstant(p1, p2 RatingRow, score1 float64) (RatingRow, RatingRow) {
	opp1 := []glicko.Opponent{glicko.Opponent{
		R:     p2.R,
		RD:    p2.RD,
		Sigma: p2.Sigma,
		SJ:    score1,
	}}
	nr1, nrd1, ns1 := glicko.Rank(p1.R, p1.RD, p1.Sigma, opp1, 0.5)

	opp2 := []glicko.Opponent{glicko.Opponent{
		R:     p1.R,
		RD:    p1.RD,
		Sigma: p1.Sigma,
		SJ:    1 - score1,
	}}
	nr2, nrd2, ns2 := glicko.Rank(p2.R, p2.RD, p2.Sigma, opp2, 0.5)

	return RatingRow{nr1, nrd1, ns1}, RatingRow{nr2, nrd2, ns2}
}

// Temporarily comment out Glicko to unblock build
/*
type Rating struct {
	R, RD, Sigma float64
}

type gOpp struct{ r, rd, sigma, sj float64 }

func (o gOpp) R() float64     { return o.r }
func (o gOpp) RD() float64    { return o.rd }
func (o gOpp) Sigma() float64 { return o.sigma }
func (o gOpp) SJ() float64    { return o.sj }

func (s *Store) UpdateGlickoInstant(p1, p2 Rating, score1, tau float64) (Rating, Rating) {
	opps1 := []jl.Opponent{gOpp{r: p2.R, rd: p2.RD, sigma: p2.Sigma, sj: score1}}
	nr1, nrd1, ns1 := jl.Rank(p1.R, p1.RD, p1.Sigma, opps1, tau)

	opps2 := []jl.Opponent{gOpp{r: p1.R, rd: p1.RD, sigma: p1.Sigma, sj: 1 - score1}}
	nr2, nrd2, ns2 := jl.Rank(p2.R, p2.RD, p2.Sigma, opps2, tau)

	return Rating{nr1, nrd1, ns1}, Rating{nr2, nrd2, ns2}
}

func (s *Store) UpdateRatings(matchID string) error {
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

	var whiteRow, blackRow Rating
	err = s.DB.QueryRow(context.Background(), "SELECT rating, rd, volatility FROM ratings WHERE user_id = $1", white).Scan(&whiteRow.R, &whiteRow.RD, &whiteRow.Sigma)
	if err != nil {
		whiteRow = Rating{R: 1500, RD: 350, Sigma: 0.06}
	}
	err = s.DB.QueryRow(context.Background(), "SELECT rating, rd, volatility FROM ratings WHERE user_id = $1", black).Scan(&blackRow.R, &blackRow.RD, &blackRow.Sigma)
	if err != nil {
		blackRow = Rating{R: 1500, RD: 350, Sigma: 0.06}
	}

	newWhite, newBlack := s.UpdateGlickoInstant(whiteRow, blackRow, whiteScore, 0.5)

	_, err = s.DB.Exec(context.Background(), "INSERT INTO ratings (user_id, rating, rd, volatility, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (user_id) DO UPDATE SET rating = $2, rd = $3, volatility = $4, updated_at = NOW()", white, newWhite.R, newWhite.RD, newWhite.Sigma)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(context.Background(), "INSERT INTO ratings (user_id, rating, rd, volatility, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (user_id) DO UPDATE SET rating = $2, rd = $3, volatility = $4, updated_at = NOW()", black, newBlack.R, newBlack.RD, newBlack.Sigma)
	return err
}
*/

// Placeholder for UpdateRatings without Glicko
func (s *Store) UpdateRatings(matchID string) error {
	// TODO: Implement simple Elo or skip for now
	return nil
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
