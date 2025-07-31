package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/takako/openai-go-demo/graph"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get API keys
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	
	serpAPIKey := os.Getenv("SERPAPI_KEY")
	if serpAPIKey == "" {
		log.Println("⚠️  SERPAPI_KEY not found - will use simulated search")
	} else {
		log.Println("✅ SerpAPI configured - real web search enabled")
	}

	// Create graph engine
	engine, err := graph.NewEngine(apiKey, serpAPIKey)
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Create context
	ctx := context.Background()

	fmt.Println("🤖 LangChainGo Research Assistant")
	fmt.Println("================================")
	fmt.Println("I can help you research topics, answer questions, or just chat!")
	fmt.Println("Commands: 'exit' to quit, 'stream' to toggle streaming mode, Ctrl+C to force stop")
	fmt.Println()

	// Interactive mode
	scanner := bufio.NewScanner(os.Stdin)
	streamingMode := false

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		
		// Handle special commands
		switch input {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "stream":
			streamingMode = !streamingMode
			fmt.Printf("Streaming mode: %v\n", streamingMode)
			continue
		case "":
			continue
		}

		// Execute the graph
		fmt.Println("\n🔄 Processing your request...")
		
		if streamingMode {
			// Execute with streaming updates
			updates := make(chan graph.GraphUpdate, 10)
			
			// Start a goroutine to handle updates
			go func() {
				var currentNodeOutput strings.Builder
				var currentNode string
				
				for update := range updates {
					switch update.Type {
					case "node_start":
						fmt.Printf("📍 %s: Starting...\n", update.Node)
						currentNode = update.Node
						currentNodeOutput.Reset()
					case "streaming_chunk":
						// Real-time streaming output
						if update.Node == currentNode {
							fmt.Print(update.Chunk)
							currentNodeOutput.WriteString(update.Chunk)
						}
					case "node_complete":
						fmt.Printf("\n✅ %s: Completed\n", update.Node)
					case "error":
						fmt.Printf("❌ Error in %s: %v\n", update.Node, update.Error)
					}
				}
			}()
			
			result, err := engine.StreamExecute(ctx, input, updates)
			if err != nil {
				fmt.Printf("\n❌ Execution failed: %v\n", err)
			} else {
				displayResult(result)
			}
		} else {
			// Execute without streaming
			result, err := engine.Execute(ctx, input)
			if err != nil {
				fmt.Printf("\n❌ Execution failed: %v\n", err)
			} else {
				displayResult(result)
			}
		}
		
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
}

func displayResult(result *graph.ExecutionResult) {
	fmt.Printf("\n⏱️  Execution time: %v\n", result.ExecutionTime)
	fmt.Printf("📊 Steps executed: %d\n", result.StepsExecuted)
	fmt.Printf("🛤️  Path: %s\n", strings.Join(result.Path, " → "))
	
	state := result.FinalState
	
	// Display based on intent
	switch state.GetIntent() {
	case "research":
		fmt.Printf("\n📚 Research Report on: %s\n", state.Topic)
		fmt.Println("=" + strings.Repeat("=", len(state.Topic)+21))
		fmt.Println(state.Report)
		
		// Show search queries used
		if len(state.SearchQueries) > 0 {
			fmt.Println("\n🔍 Search queries used:")
			for i, query := range state.SearchQueries {
				fmt.Printf("   %d. %s\n", i+1, query)
			}
		}
		
	case "qa":
		fmt.Println("\n💬 Answer:")
		fmt.Println(state.Report)
		
	case "chat":
		fmt.Println("\n💬 Response:")
		fmt.Println(state.Report)
		
	default:
		fmt.Println("\n📝 Result:")
		fmt.Println(state.Report)
	}
	
	// Show any errors
	if state.Error != nil {
		fmt.Printf("\n⚠️  Warning: %v\n", state.Error)
	}
}

func init() {
	// Set up logging
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("[Graph] ")
}