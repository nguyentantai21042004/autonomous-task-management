package webhook

// SecurityConfig defines configuration for webhook security
type SecurityConfig struct {
	Secret          string   `json:"secret"`             // GitHub/GitLab webhook secret
	AllowedIPs      []string `json:"allowed_ips"`        // Whitelist of IPs (optional)
	RateLimitPerMin int      `json:"rate_limit_per_min"` // Rate limit per source (default 60)
}
