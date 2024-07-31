# AWS Auto Refresh Cache Credentials
This library implements a wrapper around [credential_cache](https://github.com/aws/aws-sdk-go-v2/blob/main/aws/credential_cache.go).
This wrapper enables refreshing credentials without invalidating the current credentials.

## Why this library?
In one of my project, we noticed that one of our API pre-signing S3 objects was sometimes slower than expected.

As this API was called by a critical API, it had a strict latency of 200ms which was often breached.

The root cause was due to aws cache credential provider having a blocking "Retrieve" when token expired.

Given that our system is receiving low traffic, this was even more penalising for our API as token might not be refreshed on time.

## How the problem is solved?
The library cache the latest valid credential retrieved.
In parallel, it has a goroutine which refreshes the cached credential "ahead of time" (eg 1 minute before the credential expires).

## Why AWS SDK GO V2 does not support it?
There is a [pending feature request](https://github.com/aws/aws-sdk-go-v2/issues/2000) to ease the implementation.
