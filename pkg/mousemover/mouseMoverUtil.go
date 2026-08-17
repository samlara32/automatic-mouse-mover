package mousemover

import (
	"math/rand"
	"os"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func getLogger(m *MouseMover, doWriteToFile bool, filename string) *log.Logger {
	logger := log.New()
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}

	if doWriteToFile {
		_, err := os.Stat(logDir)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.Mkdir(logDir, os.ModePerm)
				if err != nil {
					log.Fatalf("error creating dir: %v", err)
				}
			}
		}

		logFile, err := os.OpenFile(logDir+"/"+filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		logger.SetOutput(logFile)
		m.logFile = logFile
	}

	return logger
}

func moveAndCheck(state *state, movePixel int, mode MovementMode, mouseMoveSuccessCh chan bool) {
	if state.override != nil { //we don't want to move mouse for tests
		mouseMoveSuccessCh <- state.override.valueToReturn
		return
	}
	currentX, currentY := robotgo.GetMousePos()

	dx := movePixel
	dy := movePixel

	switch mode {
	case ModeMicro:
		if movePixel > 0 {
			dx = 1
			dy = 1
		} else {
			dx = -1
			dy = -1
		}
	case ModeJiggle:
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		offset := r.Intn(8) + 4 // 4 to 11 px
		if movePixel < 0 {
			offset = -offset
		}
		dx = offset
		dy = offset
		if r.Intn(2) == 0 {
			dy = -dy
		}
	default:
		// Standard mode uses movePixel directly (default 10px / -10px)
	}

	moveToX := currentX + dx
	moveToY := currentY + dy
	robotgo.Move(moveToX, moveToY)

	//check if mouse moved. Sometimes mac users need to give
	//extra permission for controlling the mouse
	movedX, movedY := robotgo.GetMousePos()
	if movedX == currentX && movedY == currentY {
		mouseMoveSuccessCh <- false
	} else {
		mouseMoveSuccessCh <- true
	}
}

// getters and setters for state variable
func (s *state) isRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isAppRunning
}

func (s *state) updateRunningStatus(isRunning bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isAppRunning = isRunning
}

func (s *state) isSystemSleeping() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isSysSleeping
}

func (s *state) updateMachineSleepStatus(isSleeping bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isSysSleeping = isSleeping
}

func (s *state) getLastMouseMovedTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastMouseMovedTime
}

func (s *state) updateLastMouseMovedTime(time time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastMouseMovedTime = time
}

func (s *state) getLastErrorTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastErrorTime
}

func (s *state) updateLastErrorTime(time time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastErrorTime = time
}

func (s *state) getDidNotMoveCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.didNotMoveCount
}

func (s *state) updateDidNotMoveCount(count int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.didNotMoveCount = count
}
