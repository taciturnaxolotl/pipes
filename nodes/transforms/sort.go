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
		order = "asc"
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

		iVal := fmt.Sprintf("%v", getNestedValue(iMap, field))
		jVal := fmt.Sprintf("%v", getNestedValue(jMap, field))

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
				Placeholder: "published",
				HelpText:    "Field to sort by",
			},
			{
				Name:         "order",
				Label:        "Order",
				Type:         "select",
				Required:     false,
				DefaultValue: "asc",
				Options: []nodes.FieldOption{
					{Value: "asc", Label: "Ascending"},
					{Value: "desc", Label: "Descending"},
				},
			},
		},
	}
}
