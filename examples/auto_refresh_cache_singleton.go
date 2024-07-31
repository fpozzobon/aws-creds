package examples

import (
	"context"
	"fmt"
	"github.com/aws-creds/zaws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"sync"
)

type autoRefreshCacheSingleton struct {
	autoRefreshCache CredentialsProvider
}

var once sync.Once
var singleton = autoRefreshCacheSingleton{}

type CredentialsProvider interface {
	Retrieve(ctx context.Context) (aws.Credentials, error)
}

// GetAutoRefreshCache - Example of singleton giving back a credential provider
func GetAutoRefreshCache() CredentialsProvider {
	once.Do(func() {
		ctx := context.TODO()
		cfg, err := config.LoadDefaultConfig(
			ctx,
		)
		if err != nil {
			panic("configuration error, " + err.Error())
		}

		if provider, ok := cfg.Credentials.(zaws.CacheCredentialsProvider); ok {
			provider, _ := zaws.New(ctx, provider)
			singleton.autoRefreshCache = provider
			return
		}

		panic(fmt.Errorf("provider doesn't implement CacheCredentialsProvider"))
	})

	return singleton.autoRefreshCache
}
