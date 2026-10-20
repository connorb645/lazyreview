package claude

type event struct {
	Type     string   `json:"type"`
	Subtype  string   `json:"subtype"`
	IsError  bool     `json:"is_error"`
	Result   *string  `json:"result"`
	NumTurns int      `json:"num_turns"`
	Errors   []string `json:"errors"`
	Message  struct {
		Content []block `json:"content"`
	} `json:"message"`
}
