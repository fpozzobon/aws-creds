package zaws

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws-creds/internal/clock"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"go.uber.org/atomic"
)

func defaultPreset() []OptionsFunc {
	return []OptionsFunc{
		WithClock(clock.RealClock{}),
		WithExpiryWindow(1 * time.Minute),
		WithOnRefreshCredentialsError(func(err error) {
			log.Printf("failed to refresh credentials: %v", err)
			time.Sleep(1 * time.Minute)
		}),
	}
}

// New - creates an autoRefreshCache which starts automatically refresh of credentials
func New(ctx context.Context, provider CacheCredentialsProvider, options ...OptionsFunc) (*autoRefreshCache, error) {
	setup := Options{}
	options = append(defaultPreset(), options...)
	for _, o := range options {
		o(&setup)
	}

	autoRefreshCache := &autoRefreshCache{
		cachedCredential: provider,
		ctx:              ctx,
		invalidateToken:  sync.Mutex{},
		options:          setup,
	}

	go autoRefreshCache.start()

	return autoRefreshCache, nil
}

type CacheCredentialsProvider interface {
	Retrieve(ctx context.Context) (aws.Credentials, error)
	Invalidate()
}

type autoRefreshCache struct {
	options          Options
	cachedCredential CacheCredentialsProvider
	ctx              context.Context
	creds            atomic.Value
	invalidateToken  sync.Mutex
}

func (m *autoRefreshCache) Retrieve(ctx context.Context) (aws.Credentials, error) {
	if prevCred := m.getCreds(); prevCred != nil {
		return *prevCred, nil
	}
	c, err := m.cachedCredential.Retrieve(ctx)
	if err != nil {
		return c, err
	}
	m.setCreds(&c)
	return c, nil
}

func (m *autoRefreshCache) getCreds() *aws.Credentials {
	v := m.creds.Load()
	if v == nil {
		return nil
	}

	c := v.(*aws.Credentials)
	if c != nil && c.HasKeys() && (!c.CanExpire || c.Expires.After(m.options.Clock.Now().Round(0))) {
		return c
	}

	return nil
}

func (m *autoRefreshCache) setCreds(creds *aws.Credentials) {
	m.creds.Store(creds)
}

func (m *autoRefreshCache) refreshCredentials() error {
	c, err := m.Retrieve(m.ctx)
	if err != nil {
		return fmt.Errorf("cachedCredential.Retrieve %v", err)
	}

	if !c.CanExpire {
		return tokenNotExpireError
	}

	expireAt := m.options.Clock.RemainingDuration(c.Expires, m.options.ExpiryWindow)
	if expireAt < 0 {
		return fmt.Errorf("token ttl < beforeExpiry: %v", expireAt)
	}

	select {
	case <-m.ctx.Done():
		return closingError
	case <-m.options.Clock.After(expireAt):
		return m.swapCredential()
	}

}

var (
	closingError        = errors.New("close auto refresh")
	tokenNotExpireError = errors.New("token does not expire")
)

func (m *autoRefreshCache) swapCredential() error {
	m.invalidateToken.Lock()
	defer m.invalidateToken.Unlock()

	// to force refresh we need to invalidate the cache first
	m.cachedCredential.Invalidate()
	c, err := m.cachedCredential.Retrieve(m.ctx)
	if err != nil {
		return fmt.Errorf("m.cachedCredential.Retrieve: %v", err)
	}
	m.setCreds(&c)

	return nil
}

func (m *autoRefreshCache) start() {
	for {
		err := m.refreshCredentials()
		if err != nil {
			switch err {
			case closingError, tokenNotExpireError:
				return
			default:
				m.options.OnRefreshCredentialsError(err)
			}
		}
	}
}
