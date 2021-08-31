package common

import (
	"github.com/bingoohuang/gg/pkg/man"
	"github.com/bingoohuang/gg/pkg/osx"
	"github.com/bingoohuang/gg/pkg/ss"
	"io"
	"log"
	"os"

	"github.com/juju/ratelimit"
)

// ExpandHomeDir expands the ~ in the path if it is available.
func ExpandHomeDir(f string) string {
	return osx.ExpandHome(f)
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

	rateLimitBytes, err := man.ParseBytes(rateLimitEnv)
	if err != nil {
		log.Fatalf("failed to parse rate limit %s, error %v", rateLimitEnv, err)
	}

	// Bucket adding 100KB every second, holding max 100KB
	rateLimitBucket := ratelimit.NewBucketWithRate(float64(rateLimitBytes), int64(rateLimitBytes))
	return ratelimit.Reader(r, rateLimitBucket)
}
