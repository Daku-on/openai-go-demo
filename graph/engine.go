package graph

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Engine is the graph execution engine
type Engine struct {
	nodeRegistry *NodeRegistry
	edgeRegistry *EdgeRegistry
	flow         *GraphFlow
	maxSteps     int
}

// NewEngine creates a new graph execution engine
func NewEngine(apiKey, serpAPIKey string) (*Engine, error) {
	nodeRegistry, err := NewNodeRegistry(apiKey, serpAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create node registry: %w", err)
	}

	return &Engine{
		nodeRegistry: nodeRegistry,
		edgeRegistry: NewEdgeRegistry(),
		flow:         NewGraphFlow(),
		maxSteps:     10, // Prevent infinite loops
	}, nil
}

// ExecutionResult represents the result of graph execution
type ExecutionResult struct {
	FinalState    *AppState
	ExecutionTime time.Duration
	StepsExecuted int
	Path          []string
}

// Execute runs the graph with the given input
func (e *Engine) Execute(ctx context.Context, userInput string) (*ExecutionResult, error) {
	startTime := time.Now()
	
	// Initialize state
	state := NewAppState(userInput)
	
	// Track execution path
	var path []string
	
	// Start with the entry node
	currentNode := "classify_intent_and_topic"
	stepsExecuted := 0
	
	// Execute the graph
	for currentNode != "" && stepsExecuted < e.maxSteps {
		log.Printf("Executing node: %s", currentNode)
		path = append(path, currentNode)
		
		// Update current node in state
		state.CurrentNode = currentNode
		
		// Get and execute the node
		node, exists := e.nodeRegistry.GetNode(currentNode)
		if !exists {
			return nil, fmt.Errorf("node %s not found", currentNode)
		}
		
		// Execute the node
		if err := node(ctx, state); err != nil {
			state.SetError(err)
			return &ExecutionResult{
				FinalState:    state,
				ExecutionTime: time.Since(startTime),
				StepsExecuted: stepsExecuted,
				Path:          path,
			}, fmt.Errorf("node %s failed: %w", currentNode, err)
		}
		
		// Determine next node
		nextNode, err := e.flow.GetNextNode(currentNode, state, e.edgeRegistry)
		if err != nil {
			state.SetError(err)
			return &ExecutionResult{
				FinalState:    state,
				ExecutionTime: time.Since(startTime),
				StepsExecuted: stepsExecuted,
				Path:          path,
			}, fmt.Errorf("edge decision failed after %s: %w", currentNode, err)
		}
		
		currentNode = nextNode
		stepsExecuted++
	}
	
	// Check if we hit the step limit
	if stepsExecuted >= e.maxSteps {
		return nil, fmt.Errorf("execution exceeded maximum steps (%d)", e.maxSteps)
	}
	
	return &ExecutionResult{
		FinalState:    state,
		ExecutionTime: time.Since(startTime),
		StepsExecuted: stepsExecuted,
		Path:          path,
	}, nil
}

// StreamExecute executes the graph with streaming updates
func (e *Engine) StreamExecute(ctx context.Context, userInput string, updates chan<- GraphUpdate) (*ExecutionResult, error) {
	startTime := time.Now()
	
	// Initialize state
	state := NewAppState(userInput)
	
	// Set up streaming callback for real-time updates
	state.SetStreamingCallback(func(nodeId string, chunk string) {
		// Only skip completely empty chunks
		if len(chunk) == 0 {
			return
		}
		
		// For very long chunks, truncate but preserve structure
		displayChunk := chunk
		if len(chunk) > 2000 {
			displayChunk = chunk[:2000] + "..."
		}
		
		// Send streaming chunk as a special update type
		select {
		case updates <- GraphUpdate{
			Type:      "streaming_chunk",
			Node:      nodeId,
			State:     state.Clone(),
			Chunk:     displayChunk,
			Timestamp: time.Now(),
		}:
		case <-time.After(100 * time.Millisecond):
			// Timeout to prevent blocking
		}
	})
	
	// Track execution path
	var path []string
	
	// Start with the entry node
	currentNode := "classify_intent_and_topic"
	stepsExecuted := 0
	
	// Send initial update
	updates <- GraphUpdate{
		Type:      "start",
		Node:      currentNode,
		State:     state.Clone(),
		Timestamp: time.Now(),
	}
	
	// Execute the graph
	for currentNode != "" && stepsExecuted < e.maxSteps {
		log.Printf("Executing node: %s", currentNode)
		path = append(path, currentNode)
		
		// Update current node in state
		state.CurrentNode = currentNode
		
		// Send node start update
		updates <- GraphUpdate{
			Type:      "node_start",
			Node:      currentNode,
			State:     state.Clone(),
			Timestamp: time.Now(),
		}
		
		// Get and execute the node
		node, exists := e.nodeRegistry.GetNode(currentNode)
		if !exists {
			err := fmt.Errorf("node %s not found", currentNode)
			updates <- GraphUpdate{
				Type:      "error",
				Node:      currentNode,
				State:     state.Clone(),
				Error:     err,
				Timestamp: time.Now(),
			}
			close(updates)
			return nil, err
		}
		
		// Execute the node
		if err := node(ctx, state); err != nil {
			state.SetError(err)
			updates <- GraphUpdate{
				Type:      "error",
				Node:      currentNode,
				State:     state.Clone(),
				Error:     err,
				Timestamp: time.Now(),
			}
			close(updates)
			return &ExecutionResult{
				FinalState:    state,
				ExecutionTime: time.Since(startTime),
				StepsExecuted: stepsExecuted,
				Path:          path,
			}, fmt.Errorf("node %s failed: %w", currentNode, err)
		}
		
		// Send node complete update
		updates <- GraphUpdate{
			Type:      "node_complete",
			Node:      currentNode,
			State:     state.Clone(),
			Timestamp: time.Now(),
		}
		
		// Determine next node
		nextNode, err := e.flow.GetNextNode(currentNode, state, e.edgeRegistry)
		if err != nil {
			state.SetError(err)
			updates <- GraphUpdate{
				Type:      "error",
				Node:      currentNode,
				State:     state.Clone(),
				Error:     err,
				Timestamp: time.Now(),
			}
			close(updates)
			return &ExecutionResult{
				FinalState:    state,
				ExecutionTime: time.Since(startTime),
				StepsExecuted: stepsExecuted,
				Path:          path,
			}, fmt.Errorf("edge decision failed after %s: %w", currentNode, err)
		}
		
		currentNode = nextNode
		stepsExecuted++
	}
	
	// Check if we hit the step limit
	if stepsExecuted >= e.maxSteps {
		err := fmt.Errorf("execution exceeded maximum steps (%d)", e.maxSteps)
		updates <- GraphUpdate{
			Type:      "error",
			Node:      currentNode,
			State:     state.Clone(),
			Error:     err,
			Timestamp: time.Now(),
		}
		close(updates)
		return nil, err
	}
	
	// Send completion update
	updates <- GraphUpdate{
		Type:      "complete",
		Node:      "",
		State:     state.Clone(),
		Timestamp: time.Now(),
	}
	
	// Close the updates channel after sending all messages
	close(updates)
	
	return &ExecutionResult{
		FinalState:    state,
		ExecutionTime: time.Since(startTime),
		StepsExecuted: stepsExecuted,
		Path:          path,
	}, nil
}

// GraphUpdate represents a streaming update from the graph execution
type GraphUpdate struct {
	Type      string    // "start", "node_start", "node_complete", "error", "complete", "streaming_chunk"
	Node      string
	State     *AppState
	Error     error
	Chunk     string    // For streaming_chunk type
	Timestamp time.Time
}