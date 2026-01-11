package nodes

import (
	"context"

	"github.com/kierank/pipes/store"
)

type Node interface {
	Type() string
	Label() string
	Description() string
	Category() string // source|transform|output

	Inputs() int
	Outputs() int

	Execute(ctx context.Context, config map[string]interface{}, inputs [][]interface{}, execCtx *Context) ([]interface{}, error)

	ValidateConfig(config map[string]interface{}) error

	GetConfigSchema() *ConfigSchema
}

type ConfigSchema struct {
	Fields []ConfigField `json:"fields"`
}

type ConfigField struct {
	Name         string            `json:"name"`
	Label        string            `json:"label"`
	Type         string            `json:"type"` // text|url|number|select|textarea|checkbox
	Required     bool              `json:"required,omitempty"`
	DefaultValue interface{}       `json:"defaultValue,omitempty"`
	Options      []FieldOption     `json:"options,omitempty"`
	Placeholder  string            `json:"placeholder,omitempty"`
	HelpText     string            `json:"helpText,omitempty"`
}

type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Context struct {
	ExecutionID string
	PipeID      string
	DB          *store.DB
}

func NewContext(executionID, pipeID string, db *store.DB) *Context {
	return &Context{
		ExecutionID: executionID,
		PipeID:      pipeID,
		DB:          db,
	}
}

func (c *Context) Log(nodeID, level, message string) {
	c.DB.LogExecution(c.ExecutionID, nodeID, level, message)
}
