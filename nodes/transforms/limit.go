package transforms

import (
	"context"
	"fmt"

	"github.com/kierank/pipes/nodes"
)

type LimitNode struct{}

func (n *LimitNode) Type() string        { return "limit" }
func (n *LimitNode) Label() string       { return "Limit" }
func (n *LimitNode) Description() string { return "Limit the number of items" }
func (n *LimitNode) Category() string    { return "transform" }
func (n *LimitNode) Inputs() int         { return 1 }
func (n *LimitNode) Outputs() int        { return 1 }

func (n *LimitNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 {
		return []interface{}{}, nil
	}

	items := inputs[0]
	count, _ := config["count"].(float64)

	if count <= 0 || int(count) >= len(items) {
		return items, nil
	}

	limited := items[:int(count)]
	execCtx.Log("limit", "info", fmt.Sprintf("Limited %d -> %d items", len(items), len(limited)))

	return limited, nil
}

func (n *LimitNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *LimitNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{
			{
				Name:         "count",
				Label:        "Count",
				Type:         "number",
				Required:     true,
				DefaultValue: 10,
				HelpText:     "Maximum number of items to output",
			},
		},
	}
}
