package outputs

import (
	"context"
	"encoding/json"

	"github.com/kierank/pipes/nodes"
)

type JSONOutputNode struct{}

func (n *JSONOutputNode) Type() string        { return "json-output" }
func (n *JSONOutputNode) Label() string       { return "JSON Output" }
func (n *JSONOutputNode) Description() string { return "Output data as JSON" }
func (n *JSONOutputNode) Category() string    { return "output" }
func (n *JSONOutputNode) Inputs() int         { return 1 }
func (n *JSONOutputNode) Outputs() int        { return 0 }

func (n *JSONOutputNode) Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *nodes.Context) ([]interface{}, error) {
	if len(inputs) == 0 || len(inputs[0]) == 0 {
		execCtx.Log("json-output", "info", "No input data")
		return nil, nil
	}

	data := inputs[0]

	// Pretty print JSON
	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"count": len(data),
		"items": data,
	}, "", "  ")
	if err != nil {
		return nil, err
	}

	execCtx.Log("json-output", "info", string(jsonData))

	// Return the data (for potential chaining)
	return data, nil
}

func (n *JSONOutputNode) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (n *JSONOutputNode) GetConfigSchema() *nodes.ConfigSchema {
	return &nodes.ConfigSchema{
		Fields: []nodes.ConfigField{},
	}
}
