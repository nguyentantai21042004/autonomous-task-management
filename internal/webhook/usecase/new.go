package usecase

import (
	"autonomous-task-management/internal/webhook"
	pkgLog "autonomous-task-management/pkg/log"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

type implUseCase struct {
	config      webhook.SecurityConfig
	l           pkgLog.Logger
	rateLimiter *rateLimiter
}

func New(config webhook.SecurityConfig, l pkgLog.Logger) webhook.UseCase {
	return &implUseCase{
		config:      config,
		l:           l,
		rateLimiter: newRateLimiter(config.RateLimitPerMin),
	}
}

type rateLimiter struct {
	limiters *expirable.LRU[string, *rate.Limiter]
	rate     rate.Limit
	burst    int
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
	if requestsPerMin <= 0 {
		requestsPerMin = 60 // Default
	}
	return &rateLimiter{
		limiters: expirable.NewLRU[string, *rate.Limiter](
			1000,          // Max 1000 unique sources
			nil,           // No eviction callback
			time.Minute*5, // TTL: 5 minutes
		),
		rate:  rate.Limit(float64(requestsPerMin) / 60.0), // Per second
		burst: requestsPerMin / 10,                        // Allow burst
	}
}

func (rl *rateLimiter) Allow(key string) bool {
	limiter, ok := rl.limiters.Get(key)
	if !ok {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Add(key, limiter)
	}

	return limiter.Allow()
}
