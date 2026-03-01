package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"autonomous-task-management/internal/webhook"
)

func (uc *implUseCase) validateGitHubSignature(payload []byte, signature string) error {
	if uc.config.Secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format")
	}

	expectedSigHex := signature[7:]
	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return fmt.Errorf("invalid signature hex encoding: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(uc.config.Secret))
	mac.Write(payload)
	actualSig := mac.Sum(nil)

	if !hmac.Equal(expectedSig, actualSig) {
		return webhook.ErrInvalidSignature
	}

	return nil
}

func (uc *implUseCase) validateGitLabToken(token string) error {
	if uc.config.Secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	if token != uc.config.Secret {
		return webhook.ErrInvalidSignature
	}

	return nil
}

func (uc *implUseCase) checkRateLimit(source string) error {
	if !uc.rateLimiter.Allow(source) {
		return webhook.ErrRateLimitExceeded
	}
	return nil
}
