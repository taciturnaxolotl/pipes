package outputs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kierank/pipes/nodes"
)

type WebhookOutputNode struct{}

func (n *WebhookOutputNode) Type() string        { return "webhook-output" }
func (n *WebhookOutputNode) Label() string       { return "Webhook" }
func (n *WebhookOutputNode) Description() string { return "POST data to a webhook URL" }
func (n *WebhookOutputNode) Category() string    { return "output" }
func (n *WebhookOutputNode) Inputs() int         { return 1 }
func (n *WebhookOutputNode) Outputs() int        { return 0 }

func (n *WebhookOutputNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 || len(inputs[0]) == 0 {
		execCtx.Log("webhook-output", "info", "No input data")
		return nil, nil
	}

	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	data := inputs[0]

	payload := map[string]interface{}{
		"count": len(data),
		"items": data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pipes/1.0")

	// Add custom headers
	if headers, ok := config["headers"].(string); ok && headers != "" {
		for _, line := range strings.Split(headers, "\n") {
			if parts := strings.SplitN(strings.TrimSpace(line), ":", 2); len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}

	execCtx.Log("webhook-output", "info", fmt.Sprintf("Posted %d items to webhook (HTTP %d)", len(data), resp.StatusCode))

	return data, nil
}

func (n *WebhookOutputNode) ValidateConfig(config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("webhook URL is required")
	}
	return nil
}

func (n *WebhookOutputNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "url",
				Label:       "Webhook URL",
				Type:        "url",
				Required:    true,
				Placeholder: "https://example.com/webhook",
				HelpText:    "URL to POST data to",
			},
			{
				Name:        "headers",
				Label:       "Headers",
				Type:        "textarea",
				Required:    false,
				Placeholder: "Authorization: Bearer token",
				HelpText:    "Custom headers, one per line as Header: Value",
			},
		},
	}
}
