package mousemover

import (
	"os"
	"sync"
	"time"
)

// MovementMode defines the movement behavior of the mouse cursor
type MovementMode string

const (
	ModeStandard MovementMode = "standard" // 10px shift back and forth
	ModeMicro    MovementMode = "micro"    // 1px imperceptible nudge
	ModeJiggle   MovementMode = "jiggle"   // Random subtle human-like jiggle
)

// Config holds runtime settings for MouseMover
type Config struct {
	IntervalSeconds int          `json:"interval_seconds"`
	MovementMode    MovementMode `json:"movement_mode"`
}

// MouseMover is the main struct for the app
type MouseMover struct {
	quit    chan struct{}
	logFile *os.File
	state   *state
	config  Config
}

// state manages the internal working of the app
type state struct {
	mutex              sync.RWMutex
	isAppRunning       bool
	isSysSleeping      bool
	lastMouseMovedTime time.Time
	lastErrorTime      time.Time
	didNotMoveCount    int
	override           *override
}

// only needed for tests
type override struct {
	valueToReturn bool
}
