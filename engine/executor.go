package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kierank/pipes/nodes"
	"github.com/kierank/pipes/store"
)

type PipeConfig struct {
	Version     string       `json:"version"`
	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`
	Settings    Settings     `json:"settings"`
}

type Node struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position Position               `json:"position"`
	Config   map[string]interface{} `json:"config"`
	Label    string                 `json:"label,omitempty"`
}

type Connection struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Settings struct {
	Schedule    string       `json:"schedule,omitempty"`
	Enabled     bool         `json:"enabled"`
	Timeout     int          `json:"timeout,omitempty"`
	RetryConfig *RetryConfig `json:"retryConfig,omitempty"`
}

type RetryConfig struct {
	MaxRetries int `json:"maxRetries"`
	BackoffMs  int `json:"backoffMs"`
}

type Executor struct {
	db       *store.DB
	registry *Registry
}

func NewExecutor(db *store.DB) *Executor {
	return &Executor{
		db:       db,
		registry: NewRegistry(),
	}
}

func (e *Executor) Execute(ctx context.Context, pipeID string, triggerType string) (string, error) {
	executionID := uuid.New().String()
	startedAt := time.Now().Unix()

	// Create execution record
	if err := e.db.CreateExecution(executionID, pipeID, triggerType, startedAt); err != nil {
		return "", fmt.Errorf("create execution: %w", err)
	}

	// Fetch pipe configuration
	pipe, err := e.db.GetPipe(pipeID)
	if err != nil {
		return "", fmt.Errorf("get pipe: %w", err)
	}

	if pipe == nil {
		return "", fmt.Errorf("pipe not found: %s", pipeID)
	}

	var config PipeConfig
	if err := json.Unmarshal([]byte(pipe.Config), &config); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}

	// Execute pipeline
	itemCount, err := e.executePipeline(ctx, executionID, pipeID, &config)

	completedAt := time.Now().Unix()
	durationMs := (completedAt - startedAt) * 1000

	if err != nil {
		e.db.UpdateExecutionFailed(executionID, completedAt, durationMs, err.Error())
		return executionID, err
	}

	e.db.UpdateExecutionSuccess(executionID, completedAt, durationMs, itemCount)
	return executionID, nil
}

func (e *Executor) executePipeline(ctx context.Context, executionID, pipeID string, config *PipeConfig) (int, error) {
	// Topological sort to determine execution order
	order, err := topologicalSort(config.Nodes, config.Connections)
	if err != nil {
		return 0, fmt.Errorf("topological sort: %w", err)
	}

	nodeResults := make(map[string][]interface{})
	execCtx := nodes.NewContext(executionID, pipeID, e.db)

	for _, nodeID := range order {
		node := findNode(config.Nodes, nodeID)
		if node == nil {
			continue
		}

		// Get node implementation
		nodeImpl, err := e.registry.Get(node.Type)
		if err != nil {
			return 0, fmt.Errorf("get node type %s: %w", node.Type, err)
		}

		// Gather inputs from connected nodes
		inputs := e.gatherInputs(nodeID, config.Connections, nodeResults)

		// Execute node
		output, err := nodeImpl.Execute(ctx, node.Config, inputs, execCtx)
		if err != nil {
			e.db.LogExecution(executionID, nodeID, "error", fmt.Sprintf("Execution failed: %v", err))
			return 0, fmt.Errorf("node %s (%s): %w", nodeID, node.Type, err)
		}

		nodeResults[nodeID] = output
		
		// Log output data
		outputJSON, _ := json.Marshal(output)
		e.db.LogExecutionWithData(executionID, nodeID, "data", fmt.Sprintf("%d items", len(output)), string(outputJSON))
	}

	// Return item count from last node
	if len(order) == 0 {
		return 0, nil
	}

	lastNodeID := order[len(order)-1]
	finalOutput := nodeResults[lastNodeID]
	return len(finalOutput), nil
}

func (e *Executor) gatherInputs(nodeID string, connections []Connection, nodeResults map[string][]interface{}) [][]interface{} {
	var inputs [][]interface{}

	for _, conn := range connections {
		if conn.Target == nodeID {
			if result, ok := nodeResults[conn.Source]; ok {
				inputs = append(inputs, result)
			}
		}
	}

	return inputs
}

func topologicalSort(nodes []Node, connections []Connection) ([]string, error) {
	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)
	adjacency := make(map[string][]string)

	// Initialize
	for _, n := range nodes {
		inDegree[n.ID] = 0
		adjacency[n.ID] = []string{}
	}

	// Build graph
	for _, c := range connections {
		adjacency[c.Source] = append(adjacency[c.Source], c.Target)
		inDegree[c.Target]++
	}

	// Find sources (nodes with no incoming edges)
	queue := []string{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	sorted := []string{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, neighbor := range adjacency[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(sorted) != len(nodes) {
		return nil, fmt.Errorf("pipeline contains a cycle")
	}

	return sorted, nil
}

func findNode(nodes []Node, id string) *Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}
