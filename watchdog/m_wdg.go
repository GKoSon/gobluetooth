package watchdog

import (
	"time"
)

type Watchdog struct {
	interval time.Duration
	timer    *time.Timer
}

func New(interval time.Duration, callback func()) *Watchdog {
	w := Watchdog{
		interval: interval,
		timer:    time.AfterFunc(interval, callback),
	}
	return &w
}

func (w *Watchdog) Stop() {
	w.timer.Stop()
}
func (w *Watchdog) Reset() {
	w.timer.Reset(w.interval)
}
func (w *Watchdog) Feed() {
	w.timer.Stop()
	w.timer.Reset(w.interval)
}
