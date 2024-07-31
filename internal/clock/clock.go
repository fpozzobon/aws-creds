package clock

import "time"

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	RemainingDuration(future time.Time, jitter time.Duration) time.Duration
}

type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (r RealClock) RemainingDuration(future time.Time, jitter time.Duration) time.Duration {
	now := r.Now()
	if now.After(future) {
		return 0
	}
	return future.Sub(now) - jitter
}
