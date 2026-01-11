package transforms

import (
	"context"
	"fmt"
	"strings"

	"github.com/kierank/pipes/nodes"
)

type TruncateNode struct{}

func (n *TruncateNode) Type() string        { return "truncate" }
func (n *TruncateNode) Label() string       { return "Truncate" }
func (n *TruncateNode) Description() string { return "Limit text length in a field" }
func (n *TruncateNode) Category() string    { return "transform" }
func (n *TruncateNode) Inputs() int         { return 1 }
func (n *TruncateNode) Outputs() int        { return 1 }

func (n *TruncateNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 || len(inputs[0]) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]
	field, _ := config["field"].(string)
	maxLength := 200
	if ml, ok := config["max_length"].(float64); ok {
		maxLength = int(ml)
	}
	suffix, _ := config["suffix"].(string)
	if suffix == "" {
		suffix = "..."
	}

	if field == "" {
		return items, nil
	}

	var result []interface{}
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			result = append(result, item)
			continue
		}

		newItem := make(map[string]interface{})
		for k, v := range itemMap {
			newItem[k] = v
		}

		if val, ok := newItem[field].(string); ok {
			// Strip HTML tags first
			val = stripHTML(val)
			if len(val) > maxLength {
				// Find last space before maxLength to avoid cutting words
				cutoff := maxLength
				if idx := strings.LastIndex(val[:maxLength], " "); idx > maxLength/2 {
					cutoff = idx
				}
				newItem[field] = strings.TrimSpace(val[:cutoff]) + suffix
			} else {
				newItem[field] = val
			}
		}

		result = append(result, newItem)
	}

	execCtx.Log("truncate", "info", fmt.Sprintf("Truncated %d items", len(result)))
	return result, nil
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func (n *TruncateNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *TruncateNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "field",
				Label:       "Field",
				Type:        "text",
				Required:    true,
				Placeholder: "description",
				HelpText:    "Field to truncate",
			},
			{
				Name:         "max_length",
				Label:        "Max Length",
				Type:         "number",
				Required:     false,
				DefaultValue: 200,
				HelpText:     "Maximum character length",
			},
			{
				Name:         "suffix",
				Label:        "Suffix",
				Type:         "text",
				Required:     false,
				DefaultValue: "...",
				HelpText:     "Text to append when truncated",
			},
		},
	}
}
