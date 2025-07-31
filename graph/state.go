package graph

import (
	"sync"
)

// StreamingCallback is called when streaming chunks are received
type StreamingCallback func(nodeId string, chunk string)

// AppState represents the shared state across all nodes in the graph
type AppState struct {
	mu          sync.RWMutex
	UserInput   string            `json:"user_input"`
	Intent      string            `json:"intent"`
	Topic       string            `json:"topic"`
	SearchQueries []string        `json:"search_queries"`
	RawContents map[string]string `json:"raw_contents"`
	Report      string            `json:"report"`
	History     []Message         `json:"history"`
	Error       error             `json:"error,omitempty"`
	
	// Metadata for tracking
	CurrentNode string            `json:"current_node"`
	Metadata    map[string]interface{} `json:"metadata"`
	
	// Streaming support
	streamingCallback StreamingCallback
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Timestamp int64 `json:"timestamp"`
}

// NewAppState creates a new application state
func NewAppState(userInput string) *AppState {
	return &AppState{
		UserInput:   userInput,
		RawContents: make(map[string]string),
		Metadata:    make(map[string]interface{}),
		History:     []Message{},
	}
}

// SetStreamingCallback sets the callback for streaming updates
func (s *AppState) SetStreamingCallback(callback StreamingCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamingCallback = callback
}

// OnStreamingChunk notifies the callback about streaming chunks
func (s *AppState) OnStreamingChunk(nodeId string, chunk string) {
	s.mu.RLock()
	callback := s.streamingCallback
	s.mu.RUnlock()
	
	if callback != nil {
		callback(nodeId, chunk)
	}
}

// SetIntent safely sets the intent
func (s *AppState) SetIntent(intent string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Intent = intent
}

// SetTopic safely sets the topic
func (s *AppState) SetTopic(topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Topic = topic
}

// AddSearchQuery safely adds a search query
func (s *AppState) AddSearchQuery(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SearchQueries = append(s.SearchQueries, query)
}

// SetRawContent safely sets raw content for a source
func (s *AppState) SetRawContent(source, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RawContents[source] = content
}

// SetReport safely sets the report
func (s *AppState) SetReport(report string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Report = report
}

// SetError safely sets an error
func (s *AppState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
}

// GetIntent safely gets the intent
func (s *AppState) GetIntent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Intent
}

// GetSearchQueries safely gets the search queries
func (s *AppState) GetSearchQueries() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	queries := make([]string, len(s.SearchQueries))
	copy(queries, s.SearchQueries)
	return queries
}

// GetRawContents safely gets the raw contents
func (s *AppState) GetRawContents() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contents := make(map[string]string)
	for k, v := range s.RawContents {
		contents[k] = v
	}
	return contents
}

// Clone creates a deep copy of the state
func (s *AppState) Clone() *AppState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	clone := &AppState{
		UserInput: s.UserInput,
		Intent:    s.Intent,
		Topic:     s.Topic,
		Report:    s.Report,
		Error:     s.Error,
		CurrentNode: s.CurrentNode,
		RawContents: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}
	
	// Deep copy slices and maps
	clone.SearchQueries = make([]string, len(s.SearchQueries))
	copy(clone.SearchQueries, s.SearchQueries)
	
	for k, v := range s.RawContents {
		clone.RawContents[k] = v
	}
	
	for k, v := range s.Metadata {
		clone.Metadata[k] = v
	}
	
	clone.History = make([]Message, len(s.History))
	copy(clone.History, s.History)
	
	return clone
}