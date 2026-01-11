package transforms

import (
	"context"
	"fmt"
	"regexp"

	"github.com/kierank/pipes/nodes"
)

type RegexNode struct{}

func (n *RegexNode) Type() string        { return "regex" }
func (n *RegexNode) Label() string       { return "Regex Replace" }
func (n *RegexNode) Description() string { return "Search and replace text using regex" }
func (n *RegexNode) Category() string    { return "transform" }
func (n *RegexNode) Inputs() int         { return 1 }
func (n *RegexNode) Outputs() int        { return 1 }

func (n *RegexNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 || len(inputs[0]) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]
	field, _ := config["field"].(string)
	pattern, _ := config["pattern"].(string)
	replacement, _ := config["replacement"].(string)

	if field == "" || pattern == "" {
		return items, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	var result []interface{}
	modified := 0

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
			newVal := re.ReplaceAllString(val, replacement)
			if newVal != val {
				modified++
			}
			newItem[field] = newVal
		}

		result = append(result, newItem)
	}

	execCtx.Log("regex", "info", fmt.Sprintf("Modified %d of %d items", modified, len(result)))
	return result, nil
}

func (n *RegexNode) ValidateConfig(config map[string]interface{}) error {
	pattern, _ := config["pattern"].(string)
	if pattern != "" {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	return nil
}

func (n *RegexNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "field",
				Label:       "Field",
				Type:        "text",
				Required:    true,
				Placeholder: "title",
				HelpText:    "Field to apply regex to",
			},
			{
				Name:        "pattern",
				Label:       "Pattern",
				Type:        "text",
				Required:    true,
				Placeholder: "\\[.*?\\]",
				HelpText:    "Regex pattern to match",
			},
			{
				Name:        "replacement",
				Label:       "Replacement",
				Type:        "text",
				Required:    false,
				Placeholder: "",
				HelpText:    "Text to replace matches with (use $1, $2 for groups)",
			},
		},
	}
}
