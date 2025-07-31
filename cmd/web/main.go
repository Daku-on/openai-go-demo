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
	"github.com/takako/openai-go-demo/internal/config"
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
	// Load .env file first (for backward compatibility)
	loadLegacyDotEnv()
	
	// Load configuration using viper
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Log SerpAPI status
	if cfg.IsSerpAPIEnabled() {
		log.Println("✅ SerpAPI configured - real web search enabled")
	} else {
		log.Println("⚠️  SERPAPI_KEY not found - will use simulated search")
	}

	// Create graph engine
	engine, err := graph.NewEngine(cfg.OpenAI.APIKey, cfg.SerpAPI.APIKey)
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Static file server for CSS/JS assets
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("../../web/static/css/"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("../../web/static/js/"))))
	
	// Main routes
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, engine)
	})

	fmt.Printf("🌐 Web Server starting on http://localhost:%s\n", cfg.Server.Port)
	fmt.Printf("📊 Graph Visualizer: http://localhost:%s\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}

// loadLegacyDotEnv maintains backward compatibility with existing .env setup
func loadLegacyDotEnv() {
	// Try loading from project root paths for backward compatibility
	possiblePaths := []string{
		".env",
		"../.env", 
		"../../.env",
	}
	
	for _, envPath := range possiblePaths {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded .env from: %s", envPath)
			return
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	// Serve the new modular index.html
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..")
	htmlPath := filepath.Join(projectRoot, "web", "static", "index.html")
	
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