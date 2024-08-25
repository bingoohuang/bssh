package common

import (
	"io"
	"log"

	"github.com/bingoohuang/ngg/ss"
	"github.com/juju/ratelimit"
)

func CreateRateLimit(r io.Reader) io.Reader {
	// 从环境变量里获取每秒限速值，e.g. 10K
	rateLimitBytes, err := ss.GetenvBytes("RATELIMIT", 0)
	if err != nil {
		log.Fatalf("failed to parse rate limit error %v", err)
	}

	if rateLimitBytes <= 0 {
		return r
	}

	// Bucket adding 100KB every second, holding max 100KB
	rateLimitBucket := ratelimit.NewBucketWithRate(float64(rateLimitBytes), int64(rateLimitBytes))
	return ratelimit.Reader(r, rateLimitBucket)
}
