package common

import (
	"io"
	"log"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/juju/ratelimit"
	homedir "github.com/mitchellh/go-homedir"
)

// ExpandHomeDir expands the ~ in the path if it is available.
func ExpandHomeDir(f string) string {
	if s, err := homedir.Expand(f); err == nil {
		return s
	}

	return f
}

// Contains tells if s contains element e.
func Contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}

	return false
}

func CreateRateLimit(r io.Reader) io.Reader {
	// 从环境变量里获取每秒限速值，10K
	rateLimitEnv := os.Getenv("RATELIMIT")
	if rateLimitEnv == "" {
		return r
	}

	rateLimitBytes, err := humanize.ParseBytes(rateLimitEnv)
	if err != nil {
		log.Fatalf("failed to parse rate limit %s, error %v", rateLimitEnv, err)
	}

	// Bucket adding 100KB every second, holding max 100KB
	rateLimitBucket := ratelimit.NewBucketWithRate(float64(rateLimitBytes), int64(rateLimitBytes))
	return ratelimit.Reader(r, rateLimitBucket)
}
