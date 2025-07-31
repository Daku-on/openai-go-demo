package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SerpAPIClient handles Google search via SerpAPI
type SerpAPIClient struct {
	apiKey string
	client *http.Client
}

// NewSerpAPIClient creates a new SerpAPI client
func NewSerpAPIClient(apiKey string) *SerpAPIClient {
	return &SerpAPIClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

// SerpAPIResponse represents the response from SerpAPI
type SerpAPIResponse struct {
	OrganicResults []SearchResult `json:"organic_results"`
	Error          string         `json:"error,omitempty"`
}

// Search performs a Google search using SerpAPI
func (c *SerpAPIClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	// Build the API URL
	baseURL := "https://serpapi.com/search"
	params := url.Values{
		"q":      {query},
		"engine": {"google"},
		"api_key": {c.apiKey},
		"num":    {"10"}, // Get top 10 results
		"safe":   {"active"}, // Safe search
	}
	
	searchURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", "LangChainGo-Research-Assistant/1.0")
	
	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SerpAPI returned status %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse JSON response
	var serpResp SerpAPIResponse
	if err := json.Unmarshal(body, &serpResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check for API errors
	if serpResp.Error != "" {
		return nil, fmt.Errorf("SerpAPI error: %s", serpResp.Error)
	}
	
	return serpResp.OrganicResults, nil
}

// SearchAndSummarize performs search and creates a summary
func (c *SerpAPIClient) SearchAndSummarize(ctx context.Context, query string) (string, error) {
	results, err := c.Search(ctx, query)
	if err != nil {
		return "", err
	}
	
	if len(results) == 0 {
		return fmt.Sprintf("No search results found for: %s", query), nil
	}
	
	// Create a summary from the search results
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	
	// Include top results
	maxResults := 5
	if len(results) < maxResults {
		maxResults = len(results)
	}
	
	for i, result := range results[:maxResults] {
		summary.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, result.Title))
		summary.WriteString(fmt.Sprintf("   %s\n", result.Snippet))
		summary.WriteString(fmt.Sprintf("   Source: %s\n\n", result.Link))
	}
	
	return summary.String(), nil
}