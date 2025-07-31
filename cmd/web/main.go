package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/takako/openai-go-demo/graph"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type WebSocketMessage struct {
	Type  string `json:"type"`
	Query string `json:"query,omitempty"`
}

type WebSocketResponse struct {
	Type      string `json:"type"`
	Node      string `json:"node,omitempty"`
	Chunk     string `json:"chunk,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	// Load environment variables from project root
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..")
	envPath := filepath.Join(projectRoot, ".env")
	
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found at %s", envPath)
		// Try loading from current directory as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found in current directory either")
		}
	}

	// Get API keys
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" || apiKey == "your-api-key-here" {
		log.Fatal("OPENAI_API_KEY environment variable is required. Please set your actual API key in the .env file.")
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

	// Static file server
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, engine)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🌐 Web Server starting on http://localhost:%s\n", port)
	fmt.Printf("📊 Graph Visualizer: http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	// Serve the graph visualizer HTML
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..")
	htmlPath := filepath.Join(projectRoot, "web", "static", "graph-visualizer.html")
	
	http.ServeFile(w, r, htmlPath)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, engine *graph.Engine) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("📡 New WebSocket connection from %s", r.RemoteAddr)

	for {
		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if msg.Type == "research" && msg.Query != "" {
			go handleResearchRequest(conn, engine, msg.Query)
		}
	}
}

func handleResearchRequest(conn *websocket.Conn, engine *graph.Engine, query string) {
	ctx := context.Background()
	
	// Create a channel for graph updates
	updates := make(chan graph.GraphUpdate, 100)
	
	// Start a goroutine to forward graph updates to WebSocket
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		for update := range updates {
			wsResponse := WebSocketResponse{
				Type:      update.Type,
				Node:      update.Node,
				Chunk:     update.Chunk,
				Timestamp: update.Timestamp.UnixMilli(),
			}
			
			if update.Error != nil {
				wsResponse.Error = update.Error.Error()
			}
			
			// Send update to WebSocket client
			if err := conn.WriteJSON(wsResponse); err != nil {
				log.Printf("Failed to send WebSocket message: %v", err)
				return
			}
		}
	}()
	
	// Execute the research with streaming updates
	log.Printf("🔍 Starting research for query: %s", query)
	
	result, err := engine.StreamExecute(ctx, query, updates)
	
	// StreamExecute already closes the updates channel, so we don't send more messages
	// Just log the result
	if err != nil {
		log.Printf("Research execution failed: %v", err)
	} else {
		log.Printf("✅ Research completed in %v", result.ExecutionTime)
	}
	
	// Wait for the WebSocket goroutine to finish processing all messages
	<-done
}