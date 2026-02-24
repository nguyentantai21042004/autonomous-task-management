package webhook

// SecurityConfig holds webhook security settings
type SecurityConfig struct {
	Secret          string   // Shared secret for signature verification
	AllowedIPs      []string // IP whitelist (optional)
	RateLimitPerMin int      // Max requests per minute
}
