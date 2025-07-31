package graph

import (
	"fmt"
	"log"
)

// Edge represents a transition function between nodes
type Edge func(state *AppState) (string, error)

// EdgeRegistry manages edge logic
type EdgeRegistry struct {
	edges map[string]Edge
}

// NewEdgeRegistry creates a new edge registry
func NewEdgeRegistry() *EdgeRegistry {
	registry := &EdgeRegistry{
		edges: make(map[string]Edge),
	}

	// Register all edges
	registry.RegisterEdge("after_classify", registry.AfterClassify)
	registry.RegisterEdge("after_generate_queries", registry.AfterGenerateQueries)
	registry.RegisterEdge("after_search", registry.AfterSearch)
	registry.RegisterEdge("after_individual_search", registry.AfterIndividualSearch)
	registry.RegisterEdge("after_merge", registry.AfterMerge)
	registry.RegisterEdge("after_report", registry.AfterReport)

	return registry
}

// RegisterEdge registers an edge with a name
func (r *EdgeRegistry) RegisterEdge(name string, edge Edge) {
	r.edges[name] = edge
}

// GetEdge retrieves an edge by name
func (r *EdgeRegistry) GetEdge(name string) (Edge, bool) {
	edge, exists := r.edges[name]
	return edge, exists
}

// AfterClassify decides next node after intent classification
func (r *EdgeRegistry) AfterClassify(state *AppState) (string, error) {
	intent := state.GetIntent()
	log.Printf("Edge decision after classify: intent=%s", intent)

	switch intent {
	case "research":
		if state.Topic == "" {
			return "", fmt.Errorf("research intent requires a topic")
		}
		return "generate_search_queries", nil
	case "qa":
		return "answer_directly", nil
	case "chat":
		return "handle_chat", nil
	default:
		return "handle_chat", nil
	}
}

// AfterGenerateQueries decides next node after query generation
// Returns special format for dynamic branching: "branch:search_query"
func (r *EdgeRegistry) AfterGenerateQueries(state *AppState) (string, error) {
	if len(state.SearchQueries) == 0 {
		return "", fmt.Errorf("no search queries generated")
	}
	
	// Signal dynamic branching to engine
	return "branch:search_query", nil
}

// AfterSearch decides next node after search execution
func (r *EdgeRegistry) AfterSearch(state *AppState) (string, error) {
	if len(state.RawContents) == 0 {
		return "", fmt.Errorf("no search results collected")
	}
	return "synthesize_and_report", nil
}

// AfterIndividualSearch handles transition after individual search nodes (used in dynamic branching)
func (r *EdgeRegistry) AfterIndividualSearch(state *AppState) (string, error) {
	// This should not be called directly - individual searches are handled by the engine
	// But we keep it for completeness
	return "merge_search_results", nil
}

// AfterMerge decides next node after merging search results
func (r *EdgeRegistry) AfterMerge(state *AppState) (string, error) {
	if len(state.RawContents) == 0 {
		return "", fmt.Errorf("no search results to synthesize")
	}
	return "synthesize_and_report", nil
}

// AfterReport is the terminal edge
func (r *EdgeRegistry) AfterReport(state *AppState) (string, error) {
	// Terminal node - no next node
	return "", nil
}

// GraphFlow defines the flow structure
type GraphFlow struct {
	// Maps node names to their subsequent edge names
	NodeToEdge map[string]string
}

// NewGraphFlow creates the default graph flow
func NewGraphFlow() *GraphFlow {
	return &GraphFlow{
		NodeToEdge: map[string]string{
			"classify_intent_and_topic": "after_classify",
			"generate_search_queries":   "after_generate_queries",
			"execute_parallel_search":   "after_search",
			"merge_search_results":      "after_merge",
			"synthesize_and_report":     "after_report",
			"answer_directly":           "after_report",
			"handle_chat":               "after_report",
		},
	}
}

// GetNextNode determines the next node based on current node and state
func (f *GraphFlow) GetNextNode(currentNode string, state *AppState, edgeRegistry *EdgeRegistry) (string, error) {
	edgeName, exists := f.NodeToEdge[currentNode]
	if !exists {
		return "", nil // Terminal node
	}

	edge, exists := edgeRegistry.GetEdge(edgeName)
	if !exists {
		return "", fmt.Errorf("edge %s not found", edgeName)
	}

	return edge(state)
}