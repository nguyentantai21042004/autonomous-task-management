package agent

import (
	"autonomous-task-management/pkg/llmprovider"
	"context"
)

// Tool represents an agent tool that can be called by LLM.
type Tool interface {
	// Name returns the tool name (used in function calling).
	Name() string

	// Description returns what the tool does (for LLM).
	Description() string

	// Parameters returns JSON schema for tool parameters.
	Parameters() map[string]interface{}

	// Execute runs the tool with given parameters.
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages available tools.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools.
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToFunctionDefinitions converts tools to LLM function calling format.
func (r *ToolRegistry) ToFunctionDefinitions() []llmprovider.Tool {
	tools := make([]llmprovider.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, llmprovider.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return tools
}
