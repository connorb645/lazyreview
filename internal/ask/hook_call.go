package ask

// HookCall is one tool call as Claude Code's PreToolUse hook gives it: the
// tool's name, its input and the folder Claude works in.
type HookCall struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	Cwd       string         `json:"cwd"`
}
