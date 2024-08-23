package common

import (
	"io"
	"log"
	"os"

	"github.com/bingoohuang/ngg/rt"
	"github.com/bingoohuang/ngg/ss"
	"github.com/bingoohuang/ngg/unit"
	"github.com/juju/ratelimit"
)

// ExpandHomeDir expands the ~ in the path if it is available.
func ExpandHomeDir(f string) string {
	return rt.ExpandHome(f)
}

// Contains tells if s contains element e.
func Contains(s []string, e string) bool {
	return ss.AnyOf(e, s...)
}

func CreateRateLimit(r io.Reader) io.Reader {
	// 从环境变量里获取每秒限速值，10K
	rateLimitEnv := os.Getenv("RATELIMIT")
	if rateLimitEnv == "" {
		return r
	}

	rateLimitBytes, err := unit.ParseBytes(rateLimitEnv)
	if err != nil {
		log.Fatalf("failed to parse rate limit %s, error %v", rateLimitEnv, err)
	}

	// Bucket adding 100KB every second, holding max 100KB
	rateLimitBucket := ratelimit.NewBucketWithRate(float64(rateLimitBytes), int64(rateLimitBytes))
	return ratelimit.Reader(r, rateLimitBucket)
}
