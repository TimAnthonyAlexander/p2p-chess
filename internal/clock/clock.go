package clock

import (
	"fmt"
	"time"
)

const DriftToleranceMs = 250

func CalculateElapsed(lastTick time.Time, now time.Time, delay int) int64 {
	elapsed := now.Sub(lastTick).Milliseconds()
	if elapsed > int64(delay) {
		return elapsed - int64(delay)
	}
	return 0
}

type ClockState struct {
	MsWhite       int
	MsBlack       int
	SideToMove    string
	LastTickWhite time.Time
	LastTickBlack time.Time
	IncMs         int
	DelayMs       int
}

func UpdateClocks(state *ClockState, side string, tsServer time.Time, tsClient time.Time) (int, int, error) {
	drift := tsServer.Sub(tsClient).Milliseconds()
	if abs(drift) > DriftToleranceMs {
		// Send correction via WS
		// For now, adjust
	}
	var elapsed int64
	if side == "w" {
		elapsed = CalculateElapsed(state.LastTickWhite, tsServer, state.DelayMs)
		state.MsWhite -= int(elapsed)
		if state.MsWhite <= 0 {
			return state.MsWhite, state.MsBlack, fmt.Errorf("white timeout")
		}
		state.MsWhite += state.IncMs
		state.LastTickWhite = tsServer
		state.SideToMove = "b"
	} else {
		elapsed = CalculateElapsed(state.LastTickBlack, tsServer, state.DelayMs)
		state.MsBlack -= int(elapsed)
		if state.MsBlack <= 0 {
			return state.MsWhite, state.MsBlack, fmt.Errorf("black timeout")
		}
		state.MsBlack += state.IncMs
		state.LastTickBlack = tsServer
		state.SideToMove = "w"
	}
	return state.MsWhite, state.MsBlack, nil
}

func abs(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

// TODO: Heartbeat handling, pause/forfeit
