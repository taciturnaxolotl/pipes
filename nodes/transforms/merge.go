package transforms

import (
	"context"
	"fmt"
	"sort"

	"github.com/kierank/pipes/nodes"
)

type MergeNode struct{}

func (n *MergeNode) Type() string        { return "merge" }
func (n *MergeNode) Label() string       { return "Merge" }
func (n *MergeNode) Description() string { return "Combine multiple feeds into one" }
func (n *MergeNode) Category() string    { return "transform" }
func (n *MergeNode) Inputs() int         { return 2 }
func (n *MergeNode) Outputs() int        { return 1 }

func (n *MergeNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 {
		return []interface{}{}, nil
	}

	var merged []interface{}
	for _, input := range inputs {
		merged = append(merged, input...)
	}

	// Optionally dedupe by a field
	dedupeField, _ := config["dedupe_field"].(string)
	if dedupeField != "" {
		merged = dedupeByField(merged, dedupeField)
	}

	// Optionally sort by a field
	sortField, _ := config["sort_field"].(string)
	sortOrder, _ := config["sort_order"].(string)
	if sortField != "" {
		sortItems(merged, sortField, sortOrder == "desc")
	}

	execCtx.Log("merge", "info", fmt.Sprintf("Merged %d inputs into %d items", len(inputs), len(merged)))

	return merged, nil
}

func dedupeByField(items []interface{}, field string) []interface{} {
	seen := make(map[string]bool)
	var result []interface{}

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			result = append(result, item)
			continue
		}

		key := fmt.Sprintf("%v", itemMap[field])
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

func sortItems(items []interface{}, field string, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		iMap, iOk := items[i].(map[string]interface{})
		jMap, jOk := items[j].(map[string]interface{})
		if !iOk || !jOk {
			return false
		}

		iRaw := getNestedValue(iMap, field)
		jRaw := getNestedValue(jMap, field)

		// Try numeric comparison first (for timestamps, etc.)
		iNum, iIsNum := toFloat(iRaw)
		jNum, jIsNum := toFloat(jRaw)

		if iIsNum && jIsNum {
			if desc {
				return iNum > jNum
			}
			return iNum < jNum
		}

		// Fall back to string comparison
		iVal := fmt.Sprintf("%v", iRaw)
		jVal := fmt.Sprintf("%v", jRaw)

		if desc {
			return iVal > jVal
		}
		return iVal < jVal
	})
}

func (n *MergeNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *MergeNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "dedupe_field",
				Label:       "Dedupe Field",
				Type:        "text",
				Required:    false,
				Placeholder: "link",
				HelpText:    "Remove duplicates based on this field (e.g., link, guid)",
			},
			{
				Name:        "sort_field",
				Label:       "Sort By",
				Type:        "text",
				Required:    false,
				Placeholder: "published_at",
				HelpText:    "Field to sort merged results by (use published_at for date sorting)",
			},
			{
				Name:     "sort_order",
				Label:    "Sort Order",
				Type:     "select",
				Required: false,
				Options: []nodes.FieldOption{
					{Value: "desc", Label: "Newest First"},
					{Value: "asc", Label: "Oldest First"},
				},
			},
		},
	}
}
