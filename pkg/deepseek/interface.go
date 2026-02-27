package deepseek

import "context"

// IDeepSeek defines the interface for DeepSeek LLM client
type IDeepSeek interface {
	GenerateContent(ctx context.Context, req *Request) (*Response, error)
}
