package mock

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"time"
)

type MockClock struct {
	MNow               func() time.Time
	MAfter             func(d time.Duration) <-chan time.Time
	MRemainingDuration func(future time.Time, jitter time.Duration) time.Duration
}

func (m MockClock) Now() time.Time                         { return m.MNow() }
func (m MockClock) After(d time.Duration) <-chan time.Time { return m.MAfter(d) }
func (m MockClock) RemainingDuration(future time.Time, jitter time.Duration) time.Duration {
	return m.MRemainingDuration(future, jitter)
}

type MockCacheProvider struct {
	MRetrieve   func(ctx context.Context) (aws.Credentials, error)
	MInvalidate func()
}

// Retrieve delegates to the function value the CredentialsProviderFunc wraps.
func (m MockCacheProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return m.MRetrieve(ctx)
}

func (m MockCacheProvider) Invalidate() {
	m.MInvalidate()
}
