//go:build wasm

package main

import (
	"context"
	"syscall/js"
	"time"

	"github.com/takako/openai-go-demo/graph"
)

var engine *graph.Engine

// Global state for tracking execution
type ExecutionState struct {
	IsRunning     bool
	CurrentNode   string
	StartTime     time.Time
	StreamingData []string
}

var execState = &ExecutionState{}

func main() {
	// Initialize the Go WASM runtime
	js.Global().Set("goReady", js.ValueOf(true))
	
	// Export functions to JavaScript
	js.Global().Set("initializeEngine", js.FuncOf(initializeEngine))
	js.Global().Set("startResearch", js.FuncOf(startResearch))
	js.Global().Set("getExecutionState", js.FuncOf(getExecutionState))
	
	// Keep the program running
	select {}
}

// initializeEngine initializes the graph engine with API keys
func initializeEngine(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "OpenAI API key is required",
		})
	}
	
	apiKey := args[0].String()
	serpAPIKey := ""
	
	if len(args) > 1 {
		serpAPIKey = args[1].String()
	}
	
	var err error
	engine, err = graph.NewEngine(apiKey, serpAPIKey)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
	}
	
	return js.ValueOf(map[string]interface{}{
		"success": true,
		"message": "エンジンが初期化されました",
	})
}

// startResearch starts a research query
func startResearch(this js.Value, args []js.Value) interface{} {
	if engine == nil {
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "エンジンが初期化されていません",
		})
	}
	
	if execState.IsRunning {
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "既に実行中です",
		})
	}
	
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "クエリが必要です",
		})
	}
	
	query := args[0].String()
	callback := args[1] // JavaScript callback function
	
	// Reset execution state
	execState.IsRunning = true
	execState.StartTime = time.Now()
	execState.StreamingData = make([]string, 0)
	
	// Execute research in a goroutine
	go func() {
		defer func() {
			execState.IsRunning = false
		}()
		
		ctx := context.Background()
		updates := make(chan graph.GraphUpdate, 100)
		
		// Handle updates in a separate goroutine
		go func() {
			for update := range updates {
				execState.CurrentNode = update.Node
				
				// Convert update to JavaScript object
				updateJS := map[string]interface{}{
					"type":      update.Type,
					"node":      update.Node,
					"chunk":     update.Chunk,
					"timestamp": update.Timestamp.UnixMilli(),
				}
				
				if update.Error != nil {
					updateJS["error"] = update.Error.Error()
				}
				
				// Add to streaming data
				if update.Chunk != "" {
					execState.StreamingData = append(execState.StreamingData, update.Chunk)
				}
				
				// Call JavaScript callback
				if callback.Type() == js.TypeFunction {
					callback.Invoke(js.ValueOf(updateJS))
				}
			}
		}()
		
		// Execute the research
		result, err := engine.StreamExecute(ctx, query, updates)
		if err != nil {
			if callback.Type() == js.TypeFunction {
				callback.Invoke(js.ValueOf(map[string]interface{}{
					"type":  "error",
					"error": err.Error(),
				}))
			}
			return
		}
		
		// Send final result
		if callback.Type() == js.TypeFunction {
			callback.Invoke(js.ValueOf(map[string]interface{}{
				"type":           "complete",
				"report":         result.FinalState.Report,
				"executionTime":  result.ExecutionTime.Milliseconds(),
				"stepsExecuted":  result.StepsExecuted,
				"path":           result.Path,
			}))
		}
	}()
	
	return js.ValueOf(map[string]interface{}{
		"success": true,
		"message": "調査を開始しました",
	})
}

// getExecutionState returns current execution state
func getExecutionState(this js.Value, args []js.Value) interface{} {
	var elapsedTime int64
	if execState.IsRunning {
		elapsedTime = time.Since(execState.StartTime).Milliseconds()
	}
	
	return js.ValueOf(map[string]interface{}{
		"isRunning":     execState.IsRunning,
		"currentNode":   execState.CurrentNode,
		"elapsedTime":   elapsedTime,
		"streamingData": execState.StreamingData,
	})
}