package transforms

import (
	"context"
	"fmt"
	"sort"

	"github.com/kierank/pipes/nodes"
)

type SortNode struct{}

func (n *SortNode) Type() string        { return "sort" }
func (n *SortNode) Label() string       { return "Sort" }
func (n *SortNode) Description() string { return "Sort items by a field" }
func (n *SortNode) Category() string    { return "transform" }
func (n *SortNode) Inputs() int         { return 1 }
func (n *SortNode) Outputs() int        { return 1 }

func (n *SortNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]
	field, _ := config["field"].(string)
	order, _ := config["order"].(string)

	if field == "" {
		return items, nil
	}

	if order == "" {
		order = "desc"
	}

	// Create a sortable slice
	sorted := make([]interface{}, len(items))
	copy(sorted, items)

	sort.SliceStable(sorted, func(i, j int) bool {
		iMap, iOk := sorted[i].(map[string]interface{})
		jMap, jOk := sorted[j].(map[string]interface{})

		if !iOk || !jOk {
			return false
		}

		iRaw := getNestedValue(iMap, field)
		jRaw := getNestedValue(jMap, field)

		// Try numeric comparison first (for timestamps, etc.)
		iNum, iIsNum := toFloat(iRaw)
		jNum, jIsNum := toFloat(jRaw)

		if iIsNum && jIsNum {
			if order == "desc" {
				return iNum > jNum
			}
			return iNum < jNum
		}

		// Fall back to string comparison
		iVal := fmt.Sprintf("%v", iRaw)
		jVal := fmt.Sprintf("%v", jRaw)

		if order == "desc" {
			return iVal > jVal
		}
		return iVal < jVal
	})

	execCtx.Log("sort", "info", fmt.Sprintf("Sorted %d items by %s (%s)", len(sorted), field, order))

	return sorted, nil
}

func (n *SortNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *SortNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:        "field",
				Label:       "Field Path",
				Type:        "text",
				Required:    true,
				Placeholder: "published_at",
				HelpText:    "Field to sort by (use published_at or updated_at for date sorting)",
			},
			{
				Name:         "order",
				Label:        "Order",
				Type:         "select",
				Required:     false,
				DefaultValue: "desc",
				HelpText:     "Descending = newest first (for dates), Ascending = oldest first",
				Options: []nodes.FieldOption{
					{Value: "desc", Label: "Descending (newest first)"},
					{Value: "asc", Label: "Ascending (oldest first)"},
				},
			},
		},
	}
}
