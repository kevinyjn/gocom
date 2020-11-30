package utils

import (
	"errors"
	"time"

	"github.com/kevinyjn/gocom/logger"
)

// Timer for processs
type Timer struct {
	timer            *time.Ticker
	delay            *time.Timer
	durationMillisec int64
	cb               TimerCallback
	delegate         interface{}
}

// TimerCallback callback
type TimerCallback func(*Timer, time.Time, interface{})

// NewTimer and run the timer
func NewTimer(delayMillisec int64, durationMillisec int64, cb TimerCallback, delegate interface{}) (*Timer, error) {
	timer := &Timer{}
	err := timer.Start(delayMillisec, durationMillisec, cb, delegate)
	return timer, err
}

// Start the timer
func (p *Timer) Start(delayMillisec int64, durationMillisec int64, cb TimerCallback, delegate interface{}) error {
	if nil == cb {
		logger.Error.Printf("Starting timer while giving nil callback")
		return errors.New("Timer callback should not be empty")
	}
	if 0 == durationMillisec {
		logger.Error.Printf("Starting timer while giving zero duration")
		return errors.New("Timer duration should not be zero")
	}
	if nil != p.timer {
		p.timer.Stop()
	}
	if nil != p.delay {
		p.delay.Stop()
	}
	if 0 >= delayMillisec {
		delayMillisec = 1
	}
	p.cb = cb
	p.delegate = delegate
	p.durationMillisec = durationMillisec
	p.timer = nil
	p.delay = time.NewTimer(time.Millisecond * time.Duration(delayMillisec))
	go p.runDelay()
	return nil
}

// Stop the timer
func (p *Timer) Stop() {
	if nil != p.timer {
		p.timer.Stop()
		p.timer = nil
	}
	if nil != p.delay {
		p.delay.Stop()
		p.delay = nil
	}
}

func (p *Timer) runDelay() {
	for nil != p.delay {
		select {
		case tim := <-p.delay.C:
			p.delay.Stop()
			p.delay = nil
			p.onTrigger(tim)
			if nil != p.timer {
				p.timer.Stop()
			}
			p.timer = time.NewTicker(time.Millisecond * time.Duration(p.durationMillisec))
			go p.runTimer()
			break
		}
	}
}

func (p *Timer) runTimer() {
	for nil != p.timer {
		select {
		case tim := <-p.timer.C:
			p.onTrigger(tim)
			break
		}
	}
}

func (p *Timer) onTrigger(tim time.Time) {
	if nil != p.cb {
		go p.cb(p, tim, p.delegate)
	}
}
