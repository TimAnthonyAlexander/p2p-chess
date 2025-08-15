// Copy contents from tests.go

package referee_test

import (
	"p2p-chess/internal/referee"
	"testing"

	chess "github.com/corentings/chess/v2"
	"github.com/stretchr/testify/assert"
)

func TestValidateMove(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	uci := "e2e4"
	newFEN, err := referee.ValidateMove(fen, uci)
	assert.NoError(t, err)
	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", newFEN) // Expected FEN after e4
}

func TestInvalidMove(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	uci := "e2e5" // Illegal pawn jump
	_, err := referee.ValidateMove(fen, uci)
	assert.Error(t, err)
}

func TestCheckmateOutcome(t *testing.T) {
	fen := "rnb1kbnr/ppp3pp/8/7Q/2B5/8/PPP1PPPP/RNBQKBNR w KQkq - 0 1" // Position before checkmate
	result, err := referee.ValidateMoveWithOutcome(fen, "h5f7")
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, chess.WhiteWon, result.Outcome)
	assert.Equal(t, chess.Checkmate, result.Method)
}

// Clock tests
// ...
