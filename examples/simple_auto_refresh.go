package examples

import (
	"context"
	"github.com/aws-creds/zaws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// SimpleAutoRefresh - how to implement a simple auto refresh cache to use with S3 client
func SimpleAutoRefresh() {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(
		ctx,
	)
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	if provider, ok := cfg.Credentials.(zaws.CacheCredentialsProvider); ok {
		provider, _ := zaws.New(ctx, provider)
		// LoadDefaultConfig wrap provider with cache which we want to avoid
		cfg.Credentials = provider
	}

	// example start an S3 client with the configuration
	// cli := s3.NewFromConfig(cfg)
}
