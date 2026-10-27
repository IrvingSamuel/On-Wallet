package assistant

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const groqURL = "https://api.groq.com/openai/v1/chat/completions"

type GroqClient struct {
	apiKey       string
	chatModel    string
	visionModel  string
	httpClient   *http.Client
}

func NewGroqClient(apiKey, chatModel, visionModel string) *GroqClient {
	if chatModel == "" {
		chatModel = "openai/gpt-oss-120b"
	}
	if visionModel == "" {
		visionModel = "meta-llama/llama-4-scout-17b-16e-instruct"
	}
	return &GroqClient{
		apiKey:      apiKey,
		chatModel:   chatModel,
		visionModel: visionModel,
		httpClient:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *GroqClient) Enabled() bool {
	return c != nil && c.apiKey != ""
}

type groqMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type groqReq struct {
	Model       string    `json:"model"`
	Messages    []groqMsg `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type groqResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *GroqClient) chatMessages(system string, messages []ChatMessage) []groqMsg {
	msgs := []groqMsg{{Role: "system", Content: system}}
	for _, m := range messages {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, groqMsg{Role: role, Content: m.Content})
	}
	return msgs
}

func (c *GroqClient) Chat(ctx context.Context, system string, messages []ChatMessage) (string, error) {
	if !c.Enabled() {
		return "", ErrUnavailable
	}
	return c.complete(ctx, c.chatModel, c.chatMessages(system, messages), 0.3, 1024)
}

// ChatStream streams completion deltas via onDelta. Returns when the stream ends.
func (c *GroqClient) ChatStream(ctx context.Context, system string, messages []ChatMessage, onDelta func(string) error) error {
	if !c.Enabled() {
		return ErrUnavailable
	}
	return c.completeStream(ctx, c.chatModel, c.chatMessages(system, messages), 0.3, 1024, onDelta)
}

func (c *GroqClient) VisionOCR(ctx context.Context, mime string, data []byte, prompt string) (string, error) {
	if !c.Enabled() {
		return "", ErrUnavailable
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)
	content := []map[string]any{
		{"type": "text", "text": prompt},
		{"type": "image_url", "image_url": map[string]string{"url": dataURL}},
	}
	msgs := []groqMsg{{Role: "user", Content: content}}
	return c.complete(ctx, c.visionModel, msgs, 0.1, 2048)
}

func (c *GroqClient) complete(ctx context.Context, model string, messages []groqMsg, temp float64, maxTok int) (string, error) {
	body, err := json.Marshal(groqReq{
		Model: model, Messages: messages, Temperature: temp, MaxTokens: maxTok,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	var parsed groqResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("groq decode: %w", err)
	}
	if res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("groq %d: %s", res.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("groq: empty choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

type groqStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *GroqClient) completeStream(ctx context.Context, model string, messages []groqMsg, temp float64, maxTok int, onDelta func(string) error) error {
	body, err := json.Marshal(groqReq{
		Model: model, Messages: messages, Temperature: temp, MaxTokens: maxTok, Stream: true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
		return fmt.Errorf("groq %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, readErr := res.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				i := bytes.IndexByte(buf, '\n')
				if i < 0 {
					break
				}
				line := strings.TrimSpace(string(buf[:i]))
				buf = buf[i+1:]
				if line == "" || strings.HasPrefix(line, ":") {
					continue
				}
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "[DONE]" {
					return nil
				}
				var chunk groqStreamChunk
				if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
					continue
				}
				if chunk.Error != nil && chunk.Error.Message != "" {
					return fmt.Errorf("groq stream: %s", chunk.Error.Message)
				}
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta.Content
					if delta != "" {
						if err := onDelta(delta); err != nil {
							return err
						}
					}
				}
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
