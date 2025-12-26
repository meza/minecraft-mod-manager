package httpclient

import (
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultRateLimitInterval                = 250 * time.Millisecond
	DefaultRateLimitBurst                   = 1
	DefaultRateLimitRemainingThreshold      = 10
	DefaultRateLimitResetMaxRelativeSeconds = int64(24 * 60 * 60)
)

func DefaultLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(DefaultRateLimitInterval), DefaultRateLimitBurst)
}
