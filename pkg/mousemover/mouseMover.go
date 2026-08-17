package mousemover

import (
	"fmt"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/resousse/activity-tracker/pkg/activity"
	"github.com/resousse/activity-tracker/pkg/tracker"
)

var instance *MouseMover

const (
	timeout     = 100 //ms
	logDir      = "log"
	logFileName = "logFile-amm-5"
)

// DefaultConfig returns default runtime configuration (60s interval, standard mode)
func DefaultConfig() Config {
	return Config{
		IntervalSeconds: 60,
		MovementMode:    ModeStandard,
	}
}

// Start the main app with default settings
func (m *MouseMover) Start() {
	if m.config.IntervalSeconds <= 0 {
		m.config = DefaultConfig()
	}
	m.StartWithConfig(m.config)
}

// StartWithConfig starts the main app with custom configuration
func (m *MouseMover) StartWithConfig(cfg Config) {
	if m.state != nil && m.state.isRunning() {
		return
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 60
	}
	if cfg.MovementMode == "" {
		cfg.MovementMode = ModeStandard
	}
	m.config = cfg
	m.state = &state{}
	m.quit = make(chan struct{})

	heartbeatInterval := cfg.IntervalSeconds
	workerInterval := 10
	if heartbeatInterval <= workerInterval {
		workerInterval = heartbeatInterval / 2
		if workerInterval <= 0 {
			workerInterval = 1
		}
	}

	activityTracker := &tracker.Instance{
		HeartbeatInterval: heartbeatInterval,
		WorkerInterval:    workerInterval,
	}

	heartbeatCh := activityTracker.Start()
	m.run(heartbeatCh, activityTracker)
}

func (m *MouseMover) run(heartbeatCh chan *tracker.Heartbeat, activityTracker *tracker.Instance) {
	go func() {
		state := m.state
		if state != nil && state.isRunning() {
			return
		}
		state.updateRunningStatus(true)

		logger := getLogger(m, false, logFileName) //set writeToFile=true only for debugging
		movePixel := 10
		if m.config.MovementMode == ModeMicro {
			movePixel = 1
		}

		for {
			select {
			case heartbeat := <-heartbeatCh:
				if !heartbeat.WasAnyActivity {
					if state.isSystemSleeping() {
						logger.Infof("system sleeping")
						continue
					}
					mouseMoveSuccessCh := make(chan bool)
					go moveAndCheck(state, movePixel, m.config.MovementMode, mouseMoveSuccessCh)
					select {
					case wasMouseMoveSuccess := <-mouseMoveSuccessCh:
						if wasMouseMoveSuccess {
							state.updateLastMouseMovedTime(time.Now())
							logger.Infof("Is system sleeping? : %v : moved mouse at : %v\n\n", state.isSystemSleeping(), state.getLastMouseMovedTime())
							movePixel *= -1
							state.updateDidNotMoveCount(0)
						} else {
							didNotMoveCount := state.getDidNotMoveCount()
							state.updateDidNotMoveCount(didNotMoveCount + 1)
							state.updateLastErrorTime(time.Now())
							msg := fmt.Sprintf("Mouse pointer cannot be moved at %v. Last moved at %v. Happened %v times. (Only notifies once every 24 hours.) See README for details.",
								time.Now(), state.getLastMouseMovedTime(), state.getDidNotMoveCount())
							logger.Error(msg)
							if state.getDidNotMoveCount() >= 10 && (time.Since(state.lastErrorTime).Hours() > 24) { //show only 1 error in a 24 hour window
								go func() {
									robotgo.Alert("Error with Automatic Mouse Mover", msg)
								}()
							}
						}
					case <-time.After(timeout * time.Millisecond):
						//timeout, do nothing
						logger.Errorf("timeout happened after %vms while trying to move mouse", timeout)
					}
				} else {
					logger.Infof("activity detected in the last %v seconds.", int(activityTracker.HeartbeatInterval))
					logger.Infof("Activity type:\n")
					for activityType, times := range heartbeat.ActivityMap {
						logger.Infof("activityType : %v times: %v\n", activityType, len(times))
						if activityType == activity.MachineSleep {
							state.updateMachineSleepStatus(true)
							logger.Infof("system sleep registered. Is system sleeping? : %v", state.isSystemSleeping())
							break
						} else {
							state.updateMachineSleepStatus(false)
						}
					}
					logger.Infof("\n\n\n")
				}
			case <-m.quit:
				logger.Infof("stopping mouse mover")
				state.updateRunningStatus(false)
				activityTracker.Quit()
				return
			}
		}
	}()
}

// Quit the app
func (m *MouseMover) Quit() {
	//making it idempotent
	if m != nil && m.state != nil && m.state.isRunning() {
		m.state.updateRunningStatus(false)
		m.quit <- struct{}{}
	}
	if m.logFile != nil {
		m.logFile.Close()
	}
}

// GetInstance gets the singleton instance for mouse mover app
func GetInstance() *MouseMover {
	if instance == nil {
		instance = &MouseMover{
			state:  &state{},
			config: DefaultConfig(),
		}
	}
	return instance
}
