// Copy contents from tests.go

package referee_test

import (
	"p2p-chess/internal/referee"
	"testing"

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
	fen := "rnb1kbnr/ppppqQpp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1" // Black in checkmate
	result, err := referee.ValidateMoveWithOutcome(fen, "e8d8")       // Invalid escape
	assert.Error(t, err)
	assert.False(t, result.Valid)
	// For a valid move leading to mate, but since it's checkmate, no valid moves
}

// Clock tests
// ...
