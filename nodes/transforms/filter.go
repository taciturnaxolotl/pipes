package transforms

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kierank/pipes/nodes"
)

type FilterNode struct{}

func (n *FilterNode) Type() string        { return "filter" }
func (n *FilterNode) Label() string       { return "Filter" }
func (n *FilterNode) Description() string { return "Filter items based on conditions" }
func (n *FilterNode) Category() string    { return "transform" }
func (n *FilterNode) Inputs() int         { return 1 }
func (n *FilterNode) Outputs() int        { return 1 }

func (n *FilterNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]

	field, _ := config["field"].(string)
	operator, _ := config["operator"].(string)
	value, _ := config["value"].(string)

	if field == "" || operator == "" {
		return items, nil
	}

	var filtered []interface{}
	for _, item := range items {
		if matchesFilter(item, field, operator, value) {
			filtered = append(filtered, item)
		}
	}

	execCtx.Log("filter", "info", fmt.Sprintf("Filtered %d -> %d items", len(items), len(filtered)))

	return filtered, nil
}

func matchesFilter(item interface{}, field, operator, value string) bool {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	fieldValue := getNestedValue(itemMap, field)
	fieldStr := fmt.Sprintf("%v", fieldValue)

	switch operator {
	case "contains":
		return strings.Contains(strings.ToLower(fieldStr), strings.ToLower(value))
	case "equals":
		return fieldStr == value
	case "not-equals":
		return fieldStr != value
	case "regex":
		matched, _ := regexp.MatchString(value, fieldStr)
		return matched
	default:
		return true
	}
}

func getNestedValue(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

func (n *FilterNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *FilterNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "field",
				Label:       "Field Path",
				Type:        "text",
				Required:    true,
				Placeholder: "title",
				HelpText:    "Field to filter on (use dot notation for nested: author.name)",
			},
			{
				Name:     "operator",
				Label:    "Operator",
				Type:     "select",
				Required: true,
				Options: []nodes.FieldOption{
					{Value: "contains", Label: "Contains"},
					{Value: "equals", Label: "Equals"},
					{Value: "not-equals", Label: "Not Equals"},
					{Value: "regex", Label: "Regex Match"},
				},
			},
			{
				Name:        "value",
				Label:       "Value",
				Type:        "text",
				Required:    true,
				Placeholder: "search term",
			},
		},
	}
}
