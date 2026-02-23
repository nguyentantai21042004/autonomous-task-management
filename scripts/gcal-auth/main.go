// scripts/gcal-auth/main.go
//
// Run this ONCE locally (outside Docker) to authorize Google Calendar access
// and generate token.json.
//
// Usage:
//   go run scripts/gcal-auth/main.go
//
// It will open a browser URL, you log in with your Google account,
// paste the authorization code, and token.json will be saved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func main() {
	credsPath := "google-credentials.json"
	if len(os.Args) > 1 {
		credsPath = os.Args[1]
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		log.Fatalf("Failed to read credentials file %q: %v", credsPath, err)
	}

	config, err := google.ConfigFromJSON(data, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Failed to parse credentials: %v\nMake sure %q is an OAuth Desktop App credentials file.", err, credsPath)
	}

	// Generate the auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println("=================================================================")
	fmt.Println("BƯỚC 1: Mở URL sau trong trình duyệt và đăng nhập Google Account:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("=================================================================")
	fmt.Print("BƯỚC 2: Dán authorization code từ trình duyệt vào đây rồi Enter: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Failed to read authorization code: %v", err)
	}

	ctx := context.Background()
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Failed to exchange authorization code: %v", err)
	}

	// Save token.json
	tokenPath := "token.json"
	f, err := os.OpenFile(tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to create token.json: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(tok); err != nil {
		log.Fatalf("Failed to write token.json: %v", err)
	}

	fmt.Println()
	fmt.Printf("token.json đã được lưu tại: %s\n", tokenPath)
	fmt.Println("Bây giờ restart backend để Google Calendar hoạt động:")
	fmt.Println("  docker compose restart backend")
}
