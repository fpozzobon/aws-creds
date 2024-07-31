package zaws

import (
	"context"
	"fmt"
	"github.com/aws-creds/internal/mock"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type stubCredentialsProvider struct {
	creds   aws.Credentials
	expires time.Time
	err     error

	onInvalidate func(*stubCredentialsProvider)
}

func (s *stubCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	creds := s.creds
	creds.Source = "stubCredentialsProvider"
	creds.CanExpire = !s.expires.IsZero()
	creds.Expires = s.expires

	return creds, s.err
}

func (s *stubCredentialsProvider) Invalidate() {
	s.onInvalidate(s)
}

func TestAutoRefreshCache(t *testing.T) {

	var initClockFn = func() (mock.MockClock, time.Time, chan time.Time) {
		mockedNow := time.Now()
		mockedClockAfter := make(chan time.Time, 1)

		mockedClock := mock.MockClock{}
		mockedClock.MNow = func() time.Time {
			return mockedNow
		}
		mockedClock.MRemainingDuration = func(future time.Time, jitter time.Duration) time.Duration {
			return 0
		}
		mockedClock.MAfter = func(d time.Duration) <-chan time.Time {
			return mockedClockAfter
		}

		return mockedClock, mockedNow, mockedClockAfter
	}

	var initCacheProvider = func() mock.MockCacheProvider {
		mockedCacheProvider := mock.MockCacheProvider{}
		mockedCacheProvider.MRetrieve = func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{}, nil
		}
		mockedCacheProvider.MInvalidate = func() {}
		return mockedCacheProvider
	}

	t.Run("should refresh credential if credential is expired", func(t *testing.T) {

		mockedClock, mockedNow, _ := initClockFn()

		expiredCredential := aws.Credentials{
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			CanExpire:       true,
			Expires:         mockedNow,
		}

		newCredential := aws.Credentials{
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			CanExpire:       true,
			Expires:         mockedNow.Add(10 * time.Minute),
		}

		var called bool
		mockedCacheProvider := initCacheProvider()
		mockedCacheProvider.MRetrieve = func(ctx context.Context) (aws.Credentials, error) {
			if called {
				return newCredential, nil
			}
			called = true
			return expiredCredential, nil
		}

		p, chErr := New(context.TODO(), mockedCacheProvider, WithClock(mockedClock))
		require.Empty(t, chErr)

		// first call will get the expired credentials
		actual, err := p.Retrieve(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, actual, expiredCredential)

		// second call will get the new credentials
		actual, err = p.Retrieve(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, actual, newCredential)

	})

	t.Run("should refresh credential automatically ahead of expiration", func(t *testing.T) {
		mockedClock, mockedNow, tickRefresh := initClockFn()

		expect := aws.Credentials{
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			CanExpire:       true,
			Expires:         mockedNow.Add(1 * time.Minute),
		}

		var nbCall int
		endTest := make(chan bool)
		mockedCacheProvider := initCacheProvider()
		mockedCacheProvider.MRetrieve = func(ctx context.Context) (aws.Credentials, error) {
			nbCall++
			if nbCall >= 4 {
				endTest <- true
			}
			return expect, nil
		}

		var nbInvalidate int
		mockedCacheProvider.MInvalidate = func() {
			nbInvalidate++
		}

		_, chErr := New(context.TODO(), mockedCacheProvider, WithClock(mockedClock))
		require.Empty(t, chErr)

		mockedClock.MRemainingDuration = func(future time.Time, jitter time.Duration) time.Duration {
			assert.Equal(t, expect.Expires, future)
			assert.Equal(t, 1*time.Minute, jitter)
			return 0
		}

		tickRefresh <- time.Now()
		tickRefresh <- time.Now()
		tickRefresh <- time.Now()

		select {
		case <-endTest:
			assert.Equal(t, 4, nbCall)
			assert.Equal(t, 3, nbInvalidate)
		}
	})

	t.Run("should not stop refresh when there is error", func(t *testing.T) {
		mockedClock, mockedNow, tickRefresh := initClockFn()

		expect := aws.Credentials{
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			CanExpire:       true,
			Expires:         mockedNow.Add(1 * time.Minute),
		}

		var nbCall int
		callDone := make(chan bool, 1)
		endTest := make(chan bool, 1)
		mockedCacheProvider := initCacheProvider()
		mockedCacheProvider.MRetrieve = func(ctx context.Context) (aws.Credentials, error) {
			nbCall++
			if nbCall%2 == 0 {
				return aws.Credentials{}, fmt.Errorf("an error")
			}
			callDone <- true
			if nbCall >= 4 {
				endTest <- true
			}
			return expect, nil
		}

		var nbInvalidate int
		mockedCacheProvider.MInvalidate = func() {
			nbInvalidate++
		}

		chErr := make(chan error, 1)
		var onErr = func(err error) {
			chErr <- err
		}

		_, err := New(context.TODO(), mockedCacheProvider, WithClock(mockedClock), WithOnRefreshCredentialsError(onErr))
		require.Empty(t, err)

		mockedClock.MRemainingDuration = func(future time.Time, jitter time.Duration) time.Duration {
			assert.Equal(t, expect.Expires, future)
			assert.Equal(t, 1*time.Minute, jitter)
			return 0
		}

		for i := 0; i < 4; i++ {
			tickRefresh <- time.Now()
			select {
			case <-callDone:
				assert.Equal(t, i%2, 0)
			case <-chErr:
				assert.NotEqual(t, i%2, 0)
			}
		}

		select {
		case <-endTest:
			assert.Equal(t, 5, nbCall)
			assert.Equal(t, 4, nbInvalidate)
		}
	})

}
