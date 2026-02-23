package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ngrokTunnelsResponse matches the /api/tunnels response from the ngrok local API.
type ngrokTunnelsResponse struct {
	Tunnels []ngrokTunnel `json:"tunnels"`
}

type ngrokTunnel struct {
	PublicURL string `json:"public_url"`
	Proto     string `json:"proto"`
}

// detectNgrokURL queries the ngrok local API and returns the first HTTPS tunnel URL.
// It retries up to 10 times with 3-second intervals to handle ngrok startup race conditions.
func detectNgrokURL(ctx context.Context, ngrokAPIBase string) (string, error) {
	url := ngrokAPIBase + "/api/tunnels"
	client := &http.Client{Timeout: 5 * time.Second}

	for attempt := 1; attempt <= 10; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create ngrok API request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			if attempt < 10 {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(3 * time.Second):
					continue
				}
			}
			return "", fmt.Errorf("ngrok API not reachable after 10 attempts: %w", err)
		}
		defer resp.Body.Close()

		var tunnels ngrokTunnelsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
			return "", fmt.Errorf("failed to decode ngrok API response: %w", err)
		}

		// Prefer HTTPS tunnels
		for _, t := range tunnels.Tunnels {
			if t.Proto == "https" {
				return t.PublicURL, nil
			}
		}

		// Fallback: any tunnel
		if len(tunnels.Tunnels) > 0 {
			return tunnels.Tunnels[0].PublicURL, nil
		}

		// No tunnels yet — ngrok is starting up
		if attempt < 10 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
	}

	return "", fmt.Errorf("ngrok has no active tunnels after 10 attempts")
}
