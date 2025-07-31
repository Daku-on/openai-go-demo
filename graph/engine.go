package graph

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
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
		maxSteps:     25, // Increased for dynamic branching support
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
		
		// Check for dynamic branching signal
		if strings.HasPrefix(nextNode, "branch:") {
			branchType := strings.TrimPrefix(nextNode, "branch:")
			if branchType == "search_query" {
				// Execute dynamic search branching
				if err := e.executeDynamicSearchBranching(ctx, state, updates, &path, &stepsExecuted, startTime); err != nil {
					close(updates)
					return &ExecutionResult{
						FinalState:    state,
						ExecutionTime: time.Since(startTime),
						StepsExecuted: stepsExecuted,
						Path:          path,
					}, err
				}
				// After branching, go to merge node
				currentNode = "merge_search_results"
			} else {
				err := fmt.Errorf("unknown branch type: %s", branchType)
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
		} else {
			currentNode = nextNode
		}
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

// executeDynamicSearchBranching executes individual search queries in parallel
func (e *Engine) executeDynamicSearchBranching(ctx context.Context, state *AppState, updates chan<- GraphUpdate, path *[]string, stepsExecuted *int, startTime time.Time) error {
	queries := state.GetSearchQueries()
	if len(queries) == 0 {
		return fmt.Errorf("no search queries available for branching")
	}
	
	log.Printf("Starting dynamic search branching with %d queries", len(queries))
	for i, q := range queries {
		log.Printf("Query %d: %s", i+1, q)
	}
	
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		queryId  string
		content  string
		err      error
	}, len(queries))
	
	// Execute each query as an individual node
	for i, query := range queries {
		wg.Add(1)
		
		go func(index int, searchQuery string) {
			defer wg.Done()
			
			queryId := fmt.Sprintf("search_query_%d", index+1)
			*path = append(*path, queryId)
			
			// Send node start for individual search
			updates <- GraphUpdate{
				Type:      "node_start",
				Node:      queryId,
				State:     state.Clone(),
				Timestamp: time.Now(),
			}
			
			// Create timeout context for individual search
			searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			// Perform the actual search
			var content string
			var err error
			
			if e.nodeRegistry.serpAPI != nil {
				log.Printf("Performing real search for query %d: %s", index+1, searchQuery)
				content, err = e.nodeRegistry.serpAPI.SearchAndSummarize(searchCtx, searchQuery)
			} else {
				log.Printf("Using simulated search for query %d: %s", index+1, searchQuery)
				content, err = e.nodeRegistry.simulateSearchForBranching(searchCtx, searchQuery, queryId, state)
			}
			
			// Send result to channel
			select {
			case resultsChan <- struct {
				queryId  string
				content  string
				err      error
			}{queryId, content, err}:
			case <-searchCtx.Done():
				// Timeout or cancellation
				return
			}
			
			// Send node complete for individual search
			if err != nil {
				updates <- GraphUpdate{
					Type:      "error",
					Node:      queryId,
					State:     state.Clone(),
					Error:     err,
					Timestamp: time.Now(),
				}
			} else {
				updates <- GraphUpdate{
					Type:      "node_complete",
					Node:      queryId,
					State:     state.Clone(),
					Timestamp: time.Now(),
				}
			}
		}(i, query)
	}
	
	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	
	// Collect results
	successCount := 0
	for result := range resultsChan {
		if result.err != nil {
			log.Printf("Search error for %s: %v", result.queryId, result.err)
			continue
		}
		
		// Store result in state with proper source naming
		sourceName := fmt.Sprintf("Search_%s", result.queryId)
		state.SetRawContent(sourceName, result.content)
		successCount++
	}
	
	// Dynamic branching counts as 1 step, not len(queries) steps
	*stepsExecuted += 1
	log.Printf("Dynamic branching completed: %d/%d searches successful", successCount, len(queries))
	
	if successCount == 0 {
		return fmt.Errorf("all search branches failed")
	}
	
	return nil
}