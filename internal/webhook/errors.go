package webhook

import "errors"

var (
	ErrInvalidSignature  = errors.New("webhook: invalid signature")
	ErrUnsupportedEvent  = errors.New("webhook: unsupported event type")
	ErrRateLimitExceeded = errors.New("webhook: rate limit exceeded")
)
