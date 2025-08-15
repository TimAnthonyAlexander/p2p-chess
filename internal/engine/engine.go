package engine

import (
	"fmt"

	chess "github.com/corentings/chess/v2"
)

type Engine struct {
	Game *chess.Game
}

func NewEngine(fenStr string) (*Engine, error) {
	fenOpt, err := chess.FEN(fenStr)
	if err != nil {
		return nil, err
	}
	g := chess.NewGame(fenOpt)
	return &Engine{Game: g}, nil
}

func (e *Engine) ApplyMove(uciStr string) error {
	for _, mv := range e.Game.ValidMoves() {
		if mv.String() == uciStr {
			return e.Game.Move(&mv, nil)
		}
	}
	return fmt.Errorf("illegal move: %s", uciStr)
}

func (e *Engine) GetFEN() string {
	return e.Game.Position().String()
}

// TODO: Zobrist, SAN, etc.
