package transforms

import (
	"context"
	"fmt"
	"strings"

	"github.com/kierank/pipes/nodes"
)

type MapNode struct{}

func (n *MapNode) Type() string        { return "map" }
func (n *MapNode) Label() string       { return "Map Fields" }
func (n *MapNode) Description() string { return "Rename, extract, or create new fields" }
func (n *MapNode) Category() string    { return "transform" }
func (n *MapNode) Inputs() int         { return 1 }
func (n *MapNode) Outputs() int        { return 1 }

func (n *MapNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 || len(inputs[0]) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]
	mappings, _ := config["mappings"].(string)
	keepOriginal, _ := config["keep_original"].(bool)

	if mappings == "" {
		return items, nil
	}

	// Parse mappings: "newField:sourceField, title:name"
	fieldMap := parseMappings(mappings)

	var result []interface{}
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			result = append(result, item)
			continue
		}

		var newItem map[string]interface{}
		if keepOriginal {
			newItem = make(map[string]interface{})
			for k, v := range itemMap {
				newItem[k] = v
			}
		} else {
			newItem = make(map[string]interface{})
		}

		for newField, sourceField := range fieldMap {
			if val := getNestedValue(itemMap, sourceField); val != nil {
				newItem[newField] = val
			}
		}

		result = append(result, newItem)
	}

	execCtx.Log("map", "info", fmt.Sprintf("Mapped %d items", len(result)))
	return result, nil
}

func parseMappings(s string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, ":", 2); len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

func (n *MapNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *MapNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "mappings",
				Label:       "Field Mappings",
				Type:        "textarea",
				Required:    true,
				Placeholder: "title:name, url:link, summary:description",
				HelpText:    "Map fields as newField:sourceField, separated by commas. Use dot notation for nested fields.",
			},
			{
				Name:         "keep_original",
				Label:        "Keep Original Fields",
				Type:         "checkbox",
				Required:     false,
				DefaultValue: true,
				HelpText:     "Keep all original fields in addition to mapped ones",
			},
		},
	}
}
