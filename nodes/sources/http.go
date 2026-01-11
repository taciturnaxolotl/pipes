package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kierank/pipes/nodes"
)

type HTTPSourceNode struct{}

func (n *HTTPSourceNode) Type() string        { return "http-source" }
func (n *HTTPSourceNode) Label() string       { return "HTTP/JSON" }
func (n *HTTPSourceNode) Description() string { return "Fetch data from a JSON API" }
func (n *HTTPSourceNode) Category() string    { return "source" }
func (n *HTTPSourceNode) Inputs() int         { return 0 }
func (n *HTTPSourceNode) Outputs() int        { return 1 }

func (n *HTTPSourceNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	execCtx.Log("http-source", "info", fmt.Sprintf("Fetching %s", url))

	client := &http.Client{Timeout: 30 * time.Second}
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add custom headers
	if headers, ok := config["headers"].(string); ok && headers != "" {
		for _, line := range strings.Split(headers, "\n") {
			if parts := strings.SplitN(strings.TrimSpace(line), ":", 2); len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	req.Header.Set("User-Agent", "Pipes/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Parse JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Extract items from a path if specified
	itemsPath, _ := config["items_path"].(string)
	if itemsPath != "" {
		data = extractPath(data, itemsPath)
	}

	// Convert to array
	var items []interface{}
	switch v := data.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		items = []interface{}{v}
	default:
		items = []interface{}{data}
	}

	// Apply limit
	if limit, ok := config["limit"].(float64); ok && limit > 0 && int(limit) < len(items) {
		items = items[:int(limit)]
	}

	execCtx.Log("http-source", "info", fmt.Sprintf("Retrieved %d items", len(items)))
	return items, nil
}

func extractPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else if arr, ok := current.([]interface{}); ok {
			// Try to access array by index if part is numeric
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx < len(arr) {
				current = arr[idx]
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func (n *HTTPSourceNode) ValidateConfig(config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (n *HTTPSourceNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "url",
				Label:       "URL",
				Type:        "url",
				Required:    true,
				Placeholder: "https://api.example.com/data.json",
				HelpText:    "URL of the JSON API endpoint",
			},
			{
				Name:        "items_path",
				Label:       "Items Path",
				Type:        "text",
				Required:    false,
				Placeholder: "data.items",
				HelpText:    "Dot-notation path to the array of items (e.g., results, data.posts)",
			},
			{
				Name:        "headers",
				Label:       "Headers",
				Type:        "textarea",
				Required:    false,
				Placeholder: "Authorization: Bearer token\nAccept: application/json",
				HelpText:    "Custom headers, one per line as Header: Value",
			},
			{
				Name:         "limit",
				Label:        "Limit",
				Type:         "number",
				Required:     false,
				DefaultValue: 50,
				HelpText:     "Maximum number of items",
			},
		},
	}
}
