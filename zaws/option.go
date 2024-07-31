package zaws

import (
	"github.com/aws-creds/internal/clock"
	"time"
)

// OptionsFunc is a type alias for Options functional option
type OptionsFunc func(*Options) error

// Options are discrete set of options that are valid for creating autoRefreshCache
type Options struct {
	Clock                     clock.Clock
	ExpiryWindow              time.Duration
	OnRefreshCredentialsError func(err error)
}

// WithClock - clock used to know when cache should be refreshed: `m.clock.After(expireAt)`
// default: realClock
func WithClock(clock clock.Clock) OptionsFunc {
	return func(o *Options) error {
		o.Clock = clock
		return nil
	}
}

// WithExpiryWindow - calculate expireAt: `m.clock.RemainingDuration(c.Expires, expiryWindow)`
// default: 1 minute
func WithExpiryWindow(expiryWindow time.Duration) OptionsFunc {
	return func(o *Options) error {
		o.ExpiryWindow = expiryWindow
		return nil
	}
}

// WithOnRefreshCredentialsError - callback triggered when there is an error during refresh credential
// default: sleep 1 minutes
func WithOnRefreshCredentialsError(onRefreshCredentialsError func(err error)) OptionsFunc {
	return func(o *Options) error {
		o.OnRefreshCredentialsError = onRefreshCredentialsError
		return nil
	}
}
