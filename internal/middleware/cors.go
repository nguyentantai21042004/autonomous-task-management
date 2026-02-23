package middleware

import (
	"autonomous-task-management/internal/model"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowOriginFunc  func(origin string) bool
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

var privateSubnets = []string{
	"172.16.21.0/24", // K8s cluster subnet
	"172.16.19.0/24", // Private network 1
	"192.168.1.0/24", // Private network 2
}

var productionOrigins = []string{
	"https://smap.tantai.dev",
	"https://identity-srv.tantai.dev",
	"http://smap.tantai.dev",         // For testing/non-HTTPS
	"http://identity-srv.tantai.dev", // For testing/non-HTTPS
}

func isPrivateOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, cidr := range privateSubnets {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

func isLocalhostOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1") ||
		strings.HasPrefix(origin, "https://localhost") ||
		strings.HasPrefix(origin, "https://127.0.0.1")
}

// DefaultCORSConfig returns environment-aware CORS configuration.
func DefaultCORSConfig(environment string) CORSConfig {
	if environment == "" {
		environment = string(model.EnvironmentProduction)
	}

	config := CORSConfig{
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"Accept",
			"X-Requested-With",
			"lang",
		},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}

	// Production: strict origin list only
	if environment == "production" {
		config.AllowedOrigins = productionOrigins
		return config
	}

	// Development/Staging: dynamic origin validation
	config.AllowOriginFunc = func(origin string) bool {
		// Allow production domains
		for _, allowed := range productionOrigins {
			if origin == allowed {
				return true
			}
		}

		// Allow localhost (any port)
		if isLocalhostOrigin(origin) {
			return true
		}

		// Allow private subnets
		if isPrivateOrigin(origin) {
			return true
		}

		return false
	}

	return config
}

func CORS(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed using dynamic validation function (if set)
		originAllowed := false
		if config.AllowOriginFunc != nil {
			originAllowed = config.AllowOriginFunc(origin)
			if originAllowed {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		} else if isOriginAllowed(origin, config.AllowedOrigins) {
			// Fall back to static origin list
			c.Header("Access-Control-Allow-Origin", origin)
			originAllowed = true
		} else if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
			originAllowed = true
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if len(config.AllowedMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}
			if len(config.AllowedHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}
			if len(config.ExposedHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}
			c.AbortWithStatus(204) // No Content
			return
		}

		// Set headers for actual requests
		if len(config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		}
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Next()
	}
}

// isOriginAllowed checks if the given origin is in the allowed origins list.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
