package upstream

type ChatCompletionRequest struct {
	Model              string             `json:"model"`
	Messages           []Message          `json:"messages"`
	Temperature        float64            `json:"temperature,omitempty"`
	ChatTemplateKwargs ChatTemplateKwargs `json:"chat_template_kwargs,omitempty"`
}

type ChatTemplateKwargs struct {
	EnableThinking bool `json:"enable_thinking"`
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type Choice struct {
	Message AssistantMessage `json:"message"`
}

type AssistantMessage struct {
	Content string `json:"content"`
}

type RecognizeRequest struct {
	Model    string
	Prompt   string
	ImagePath string
	MIMEType string
}

type RecognizeResult struct {
	Text string
}
