package qwen

import "time"

const (
	// DefaultModel is the default Qwen model
	DefaultModel = "qwen-plus"

	// DefaultBaseURL is the default Qwen API endpoint
	DefaultBaseURL = "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)
