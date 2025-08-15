package clock_test

import (
	"p2p-chess/internal/clock"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUpdateClocks(t *testing.T) {
	state := &clock.ClockState{
		MsWhite:       300000,
		MsBlack:       300000,
		SideToMove:    "w",
		LastTickWhite: time.Now().Add(-2 * time.Second),
		LastTickBlack: time.Now().Add(-10 * time.Second),
		IncMs:         3000,
		DelayMs:       0,
	}
	tsServer := time.Now()
	tsClient := tsServer.Add(-100 * time.Millisecond) // Small drift

	newWhite, newBlack, err := clock.UpdateClocks(state, "w", tsServer, tsClient)
	assert.NoError(t, err)
	assert.Greater(t, newWhite, 299000) // Account for elapsed ~2000, net +1000
	assert.Equal(t, newBlack, 300000)
	assert.Equal(t, state.SideToMove, "b")
}

// Test timeout, drift > tolerance, etc.
