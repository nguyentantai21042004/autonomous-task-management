package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

// SecurityValidator validates webhook requests
type SecurityValidator struct {
	config      SecurityConfig
	rateLimiter *rateLimiter
}

func NewSecurityValidator(config SecurityConfig) *SecurityValidator {
	return &SecurityValidator{
		config:      config,
		rateLimiter: newRateLimiter(config.RateLimitPerMin),
	}
}

// ValidateGitHubSignature verifies GitHub webhook signature
func (v *SecurityValidator) ValidateGitHubSignature(payload []byte, signature string) error {
	if v.config.Secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// GitHub sends signature as "sha256=<hex>"
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format")
	}

	expectedSigHex := signature[7:] // Remove "sha256=" prefix

	// Decode hex to bytes for more secure comparison
	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return fmt.Errorf("invalid signature hex encoding: %w", err)
	}

	// Calculate HMAC
	mac := hmac.New(sha256.New, []byte(v.config.Secret))
	mac.Write(payload)
	actualSig := mac.Sum(nil)

	// Constant-time comparison on raw bytes
	if !hmac.Equal(expectedSig, actualSig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// ValidateGitLabToken verifies GitLab webhook token
func (v *SecurityValidator) ValidateGitLabToken(token string) error {
	if v.config.Secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	if token != v.config.Secret {
		return fmt.Errorf("invalid token")
	}

	return nil
}

// ValidateIPAddress checks if request IP is whitelisted
func (v *SecurityValidator) ValidateIPAddress(r *http.Request) error {
	if len(v.config.AllowedIPs) == 0 {
		return nil // No IP restriction
	}

	// Extract IP from request
	ip := extractIP(r)

	// Check against whitelist
	for _, allowedIP := range v.config.AllowedIPs {
		if ip == allowedIP {
			return nil
		}

		// Check CIDR range
		if strings.Contains(allowedIP, "/") {
			_, ipNet, err := net.ParseCIDR(allowedIP)
			if err != nil {
				continue
			}
			if ipNet.Contains(net.ParseIP(ip)) {
				return nil
			}
		}
	}

	return fmt.Errorf("IP %s not whitelisted", ip)
}

// CheckRateLimit enforces rate limiting
func (v *SecurityValidator) CheckRateLimit(source string) error {
	return v.rateLimiter.Allow(source)
}

// extractIP extracts client IP from request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// rateLimiter is a production-grade rate limiter with auto-cleanup
type rateLimiter struct {
	limiters *expirable.LRU[string, *rate.Limiter]
	rate     rate.Limit
	burst    int
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
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

func (rl *rateLimiter) Allow(key string) error {
	limiter, ok := rl.limiters.Get(key)
	if !ok {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Add(key, limiter)
	}

	if !limiter.Allow() {
		return fmt.Errorf("rate limit exceeded for %s", key)
	}
	return nil
}
