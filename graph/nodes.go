package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/takako/openai-go-demo/tools"
)

// Node represents a processing node in the graph
type Node func(ctx context.Context, state *AppState) error

// NodeRegistry holds all available nodes
type NodeRegistry struct {
	nodes     map[string]Node
	llm       llms.Model
	serpAPI   *tools.SerpAPIClient
}

// NewNodeRegistry creates a new node registry with an LLM
func NewNodeRegistry(apiKey, serpAPIKey string) (*NodeRegistry, error) {
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithModel("gpt-4o-2024-08-06"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// Initialize SerpAPI client (optional)
	var serpAPI *tools.SerpAPIClient
	if serpAPIKey != "" {
		serpAPI = tools.NewSerpAPIClient(serpAPIKey)
	}

	registry := &NodeRegistry{
		nodes:   make(map[string]Node),
		llm:     llm,
		serpAPI: serpAPI,
	}

	// Register all nodes
	registry.RegisterNode("classify_intent_and_topic", registry.ClassifyIntentAndTopic)
	registry.RegisterNode("generate_search_queries", registry.GenerateSearchQueries)
	registry.RegisterNode("execute_parallel_search", registry.ExecuteParallelSearch)
	registry.RegisterNode("merge_search_results", registry.MergeSearchResults)
	registry.RegisterNode("synthesize_and_report", registry.SynthesizeAndReport)
	registry.RegisterNode("answer_directly", registry.AnswerDirectly)
	registry.RegisterNode("handle_chat", registry.HandleChat)

	return registry, nil
}

// RegisterNode registers a node with a name
func (r *NodeRegistry) RegisterNode(name string, node Node) {
	r.nodes[name] = node
}

// GetNode retrieves a node by name
func (r *NodeRegistry) GetNode(name string) (Node, bool) {
	node, exists := r.nodes[name]
	return node, exists
}

// ClassifyIntentAndTopic determines user intent and extracts the topic  
func (r *NodeRegistry) ClassifyIntentAndTopic(ctx context.Context, state *AppState) error {
	log.Printf("DEBUG: Classifying input: '%s'", state.UserInput)
	
	// First, try keyword-based classification for reliable results
	keywordResult := r.classifyByKeywords(state.UserInput)
	log.Printf("DEBUG: Keyword classification result: intent=%s, topic=%s", keywordResult.Intent, keywordResult.Topic)
	
	// If keyword classification returns research, ALWAYS use it (highest priority)
	if keywordResult.Intent == "research" {
		state.SetIntent(keywordResult.Intent)
		state.SetTopic(keywordResult.Topic)
		log.Printf("✅ Classified by keywords: intent=%s, topic=%s", keywordResult.Intent, keywordResult.Topic)
		return nil
	}
	
	// If keyword classification returns chat, use it too
	if keywordResult.Intent == "chat" {
		state.SetIntent(keywordResult.Intent)
		state.SetTopic(keywordResult.Topic)
		log.Printf("✅ Classified by keywords: intent=%s, topic=%s", keywordResult.Intent, keywordResult.Topic)
		return nil
	}
	
	// For ambiguous cases, ask LLM (but without streaming to avoid JSON parsing issues)
	prompt := fmt.Sprintf(`Analyze the following user input and determine the intent:

INTENT CLASSIFICATION RULES:
- "research": Use for ANY educational or informational request, including:
  - "教えて" (tell me about)
  - "について知りたい" (want to know about)  
  - "調べて" (investigate/research)
  - "最新の" (latest information)
  - Questions asking for explanations, overviews, or detailed information
  - Any request that would benefit from web search and comprehensive analysis

- "qa": ONLY for simple factual questions with short answers:
  - "何時ですか" (what time is it)
  - "1+1は" (what is 1+1)
  - Basic math, definitions that don't need research

- "chat": Casual conversation, greetings, thanks, general chat

User input: "%s"

Examples:
- "Go言語について教えて" → "research" (topic: "Go言語")
- "LLMの最新動向を知りたい" → "research" (topic: "LLMの最新動向") 
- "Pythonとは何ですか" → "research" (topic: "Python")
- "今何時ですか" → "qa" (topic: "")
- "ありがとう" → "chat" (topic: "")

Respond in JSON format:
{
  "intent": "research|qa|chat",
  "topic": "extracted topic or empty string"
}`, state.UserInput)

	// NO STREAMING for classification to ensure complete JSON response
	response, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt)
	if err != nil {
		return fmt.Errorf("failed to classify intent: %w", err)
	}

	// Parse the JSON response
	var result struct {
		Intent string `json:"intent"`
		Topic  string `json:"topic"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: use keyword-based classification
		result = r.classifyByKeywords(state.UserInput)
	}

	// Additional validation: ensure "教えて" type questions go to research
	if result.Intent == "qa" {
		input := strings.ToLower(state.UserInput)
		researchKeywords := []string{
			"教えて", "おしえて", "知りたい", "について", "とは", "調べて",
			"最新", "動向", "状況", "現状", "詳しく", "説明", "解説",
			"tell me", "about", "what is", "explain", "latest", "current",
		}
		
		for _, keyword := range researchKeywords {
			if strings.Contains(input, keyword) {
				result.Intent = "research"
				if result.Topic == "" {
					// Extract potential topic from input
					result.Topic = state.UserInput
				}
				break
			}
		}
	}

	state.SetIntent(result.Intent)
	state.SetTopic(result.Topic)
	
	log.Printf("Classified intent: %s, topic: %s", result.Intent, result.Topic)
	return nil
}

// classifyByKeywords provides fallback keyword-based classification
func (r *NodeRegistry) classifyByKeywords(input string) struct {
	Intent string `json:"intent"`
	Topic  string `json:"topic"`
} {
	lower := strings.ToLower(input)
	log.Printf("DEBUG: Keyword analysis for: '%s' -> '%s'", input, lower)
	
	// Research keywords (broad match) - より包括的に
	researchKeywords := []string{
		// 日本語パターン
		"教えて", "おしえて", "知りたい", "について", "とは", "調べて",
		"最新", "動向", "状況", "現状", "詳しく", "説明", "解説", "情報",
		"どんな", "どのような", "なぜ", "なんで", "いつ", "どこ", "どうして",
		"の話", "のこと", "に関して", "関連", "特徴", "概要", "紹介",
		
		// 英語パターン
		"tell me", "about", "what is", "what are", "explain", "describe",
		"latest", "current", "how", "why", "where", "when", "who",
		"overview", "summary", "research", "investigate", "details",
		"information", "facts", "story", "background",
		
		// CM、エンタメ関連
		"cm", "コマーシャル", "広告", "テレビ", "番組", "映画", "ドラマ",
		"アニメ", "漫画", "歌", "音楽", "アーティスト", "芸能人", "タレント",
	}
	
	// Chat keywords
	chatKeywords := []string{
		"こんにちは", "hello", "hi", "ありがとう", "thank", "よろしく",
		"さようなら", "goodbye", "bye", "おつかれ", "がんばって",
	}
	
	// Simple QA keywords (very specific)
	qaKeywords := []string{
		"何時", "what time", "今日", "today", "天気", "weather",
		"計算", "calculator", "+", "-", "*", "/", "=",
	}
	
	// Check for chat first (most specific)
	for _, keyword := range chatKeywords {
		if strings.Contains(lower, keyword) {
			return struct {
				Intent string `json:"intent"`
				Topic  string `json:"topic"`
			}{"chat", ""}
		}
	}
	
	// Check for simple QA (specific cases only)
	for _, keyword := range qaKeywords {
		if strings.Contains(lower, keyword) {
			return struct {
				Intent string `json:"intent"`
				Topic  string `json:"topic"`
			}{"qa", ""}
		}
	}
	
	// Check for research (broad match)
	for _, keyword := range researchKeywords {
		if strings.Contains(lower, keyword) {
			log.Printf("DEBUG: ✅ RESEARCH keyword matched: '%s'", keyword)
			return struct {
				Intent string `json:"intent"`
				Topic  string `json:"topic"`
			}{"research", input}
		}
	}
	log.Printf("DEBUG: ❌ No research keywords matched")
	
	// Default to research for educational/informational queries
	return struct {
		Intent string `json:"intent"`
		Topic  string `json:"topic"`
	}{"research", input}
}

// GenerateSearchQueries creates multiple search queries for comprehensive research
func (r *NodeRegistry) GenerateSearchQueries(ctx context.Context, state *AppState) error {
	prompt := fmt.Sprintf(`以下のトピックについて包括的に調査するための多様な検索クエリを4-5個生成してください: "%s"

以下の観点を含めてください:
1. 基本的な紹介と概要
2. 最新の動向とニュース
3. 実用例と応用事例
4. 技術的詳細や実装方法
5. 課題と制限

JSON配列形式で検索クエリを返してください:
["クエリ1", "クエリ2", "クエリ3", "クエリ4", "クエリ5"]`, state.Topic)

	// Use streaming for real-time updates
	var response strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		response.Write(chunk)
		state.OnStreamingChunk("generate_search_queries", string(chunk))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to generate search queries: %w", err)
	}

	// Parse the JSON response using gjson library
	var queries []string
	responseStr := response.String()
	log.Printf("DEBUG: Raw LLM response for query generation: %q", responseStr)
	
	// Extract JSON array using regex and gjson
	if parsedQueries := parseQueriesWithGJson(responseStr); len(parsedQueries) > 0 {
		queries = parsedQueries
		log.Printf("DEBUG: gjson parsing successful, got %d queries", len(queries))
	} else {
		// Fallback to text extraction
		log.Printf("DEBUG: gjson parsing failed, using text extraction fallback")
		queries = extractQueriesFromText(responseStr)
	}

	// Add queries to state
	for i, query := range queries {
		query = strings.TrimSpace(query)
		if query != "" {
			log.Printf("DEBUG: Adding query %d: %q", i+1, query)
			state.AddSearchQuery(query)
		} else {
			log.Printf("DEBUG: Skipping empty query at index %d", i+1)
		}
	}

	finalQueries := state.GetSearchQueries()
	log.Printf("Generated %d search queries total", len(finalQueries))
	for i, q := range finalQueries {
		log.Printf("Final query %d: %q", i+1, q)
	}
	return nil
}

// ExecuteParallelSearch performs concurrent searches
func (r *NodeRegistry) ExecuteParallelSearch(ctx context.Context, state *AppState) error {
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		source  string
		content string
		err     error
	}, len(state.SearchQueries))

	// Execute searches in parallel with timeout protection
	for i, query := range state.SearchQueries {
		wg.Add(1)
		go func(idx int, q string) {
			defer wg.Done()
			
			// Create timeout context for individual search
			searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			// Perform real search using SerpAPI or fallback to simulation
			content, err := r.realSearch(searchCtx, q)
			
			// Safely truncate query for source name
			sourceName := q
			if len(sourceName) > 20 {
				sourceName = sourceName[:20]
			}
			
			select {
			case resultsChan <- struct {
				source  string
				content string
				err     error
			}{
				source:  fmt.Sprintf("Search_%d_%s", idx, sourceName),
				content: content,
				err:     err,
			}:
			case <-searchCtx.Done():
				// Timeout or cancellation
				return
			}
		}(i, query)
	}

	// Wait for all searches to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		if result.err != nil {
			log.Printf("Search error for %s: %v", result.source, result.err)
			continue
		}
		state.SetRawContent(result.source, result.content)
	}

	log.Printf("Collected %d search results", len(state.RawContents))
	return nil
}

// realSearch performs actual web search using SerpAPI
func (r *NodeRegistry) realSearch(ctx context.Context, query string) (string, error) {
	// If SerpAPI is available, use real search
	if r.serpAPI != nil {
		log.Printf("Performing real search for: %s", query)
		return r.serpAPI.SearchAndSummarize(ctx, query)
	}
	
	// Fallback to LLM-simulated search
	log.Printf("SerpAPI not available, falling back to simulated search for: %s", query)
	return r.simulateSearch(ctx, query)
}

// simulateSearch simulates a search operation using LLM
func (r *NodeRegistry) simulateSearch(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`以下の検索クエリに対する簡潔で事実に基づいた要約を日本語で2-3段落で提供してください: "%s"

正確で最新の情報に焦点を当ててください。技術関連の場合は、最新の開発動向も含めてください。`, query)

	// Use streaming for real-time updates
	var response strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		response.Write(chunk)
		// Note: simulateSearch doesn't have access to state, so no streaming callback here
		return nil
	}))
	if err != nil {
		return "", err
	}

	// Simulate network delay
	time.Sleep(100 * time.Millisecond)
	
	return response.String(), nil
}

// simulateSearchForBranching simulates a search operation for individual query nodes with streaming
func (r *NodeRegistry) simulateSearchForBranching(ctx context.Context, query, queryId string, state *AppState) (string, error) {
	prompt := fmt.Sprintf(`以下の検索クエリに対する簡潔で事実に基づいた要約を日本語で2-3段落で提供してください: "%s"

正確で最新の情報に焦点を当ててください。技術関連の場合は、最新の開発動向も含めてください。`, query)

	// Use streaming for real-time updates
	var response strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		response.Write(chunk)
		// Send streaming updates for this individual query node
		state.OnStreamingChunk(queryId, string(chunk))
		return nil
	}))
	if err != nil {
		return "", err
	}

	// Simulate network delay
	time.Sleep(100 * time.Millisecond)
	
	return response.String(), nil
}

// parseQueriesWithGJson extracts search queries using gjson library
func parseQueriesWithGJson(text string) []string {
	// Remove markdown code blocks first
	cleanText := regexp.MustCompile("```(?:json)?").ReplaceAllString(text, "")
	
	// Find JSON array using regex
	jsonArrayRegex := regexp.MustCompile(`\[[\s\S]*?\]`)
	matches := jsonArrayRegex.FindAllString(cleanText, -1)
	
	for _, match := range matches {
		// Try to parse with gjson
		if gjson.Valid(match) {
			var queries []string
			
			// Get array elements
			result := gjson.Parse(match)
			if result.IsArray() {
				result.ForEach(func(key, value gjson.Result) bool {
					if query := strings.TrimSpace(value.String()); len(query) > 5 {
						queries = append(queries, query)
					}
					return true // continue iteration
				})
				
				if len(queries) > 0 {
					return queries
				}
			}
		}
	}
	
	return nil
}

// extractQueriesFromText extracts search queries from text when JSON parsing fails
func extractQueriesFromText(text string) []string {
	var queries []string
	lines := strings.Split(text, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with quotes (likely search queries)
		if strings.Contains(line, "\"") && len(line) > 5 {
			// Extract content between quotes
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end != -1 && end > start {
				query := line[start+1 : end]
				query = strings.TrimSpace(query)
				if len(query) > 5 { // Only meaningful queries
					queries = append(queries, query)
				}
			}
		}
	}
	
	log.Printf("DEBUG: Text extraction found %d queries", len(queries))
	return queries
}

// MergeSearchResults merges the results from individual search branches
func (r *NodeRegistry) MergeSearchResults(ctx context.Context, state *AppState) error {
	searchResults := state.GetRawContents()
	log.Printf("Merging search results from %d individual searches", len(searchResults))
	
	// Results are already stored in state by the dynamic branching engine
	// This node just validates that we have results and logs the merge completion
	if len(searchResults) == 0 {
		return fmt.Errorf("no search results to merge")
	}
	
	log.Printf("Successfully merged %d search results", len(searchResults))
	return nil
}

// SynthesizeAndReport creates a comprehensive report from search results
func (r *NodeRegistry) SynthesizeAndReport(ctx context.Context, state *AppState) error {
	// Combine all search results
	var allContent strings.Builder
	for source, content := range state.RawContents {
		allContent.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", source, content))
	}

	prompt := fmt.Sprintf(`以下は調査レポートの例です。この例と同じ書式で「%s」に関するレポートを作成してください。

検索結果:
%s

# AI開発ツールに関する調査レポート

## 要約

AI開発の分野では新しいツールやフレームワークが急速に発展している。特にLangChainやLLMを活用した開発環境の整備が進んでおり、開発者の生産性向上に大きく寄与している。

## 主要な発見事項

1. **開発効率の向上**: AI開発ツールにより従来の3倍の開発速度を実現
2. **統合環境の充実**: LangChainを中心とした統合開発環境が普及
3. **コミュニティの活発化**: オープンソースプロジェクトが急速に成長

## 詳細分析

現在のAI開発ツール市場では、特にLangChainエコシステムが中心的な役割を果たしている。このフレームワークは複雑なAIアプリケーションの開発を大幅に簡素化し、多くの企業で採用されている。

さらに、WebSocketやストリーミング技術との組み合わせにより、リアルタイム性を重視したアプリケーションの開発も容易になっている。これらの技術的進歩により、従来は困難だった複雑なAIワークフローの実装が可能になった。

## 関連技術・概念

- **LangChain**: AI開発のためのフレームワーク
- **WebSocket**: リアルタイム通信技術
- **ストリーミング**: データのリアルタイム処理

## 推奨事項・次のステップ

- **導入検討**: 既存プロジェクトへのLangChain導入を検討する
- **技術習得**: チーム全体でのAI開発スキルの向上を図る

上記の例と同じ書式で「%s」についてのレポートを作成してください。`, state.Topic, allContent.String(), state.Topic)

	// Use streaming for real-time updates
	var report strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		report.Write(chunk)
		
		// Debug: Log chunk content to see what we're getting
		chunkStr := string(chunk)
		log.Printf("LLM chunk received: %q (length: %d)", chunkStr, len(chunkStr))
		
		state.OnStreamingChunk("synthesize_and_report", chunkStr)
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	reportStr := report.String()
	state.SetReport(reportStr)
	log.Printf("Generated report with %d characters", len(reportStr))
	return nil
}

// AnswerDirectly handles simple Q&A
func (r *NodeRegistry) AnswerDirectly(ctx context.Context, state *AppState) error {
	prompt := fmt.Sprintf("以下の質問に日本語で簡潔に回答してください: %s", state.UserInput)
	
	// Use streaming for real-time updates
	var response strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		response.Write(chunk)
		state.OnStreamingChunk("answer_directly", string(chunk))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to generate answer: %w", err)
	}

	state.SetReport(response.String())
	return nil
}

// HandleChat handles general conversation
func (r *NodeRegistry) HandleChat(ctx context.Context, state *AppState) error {
	prompt := fmt.Sprintf("以下のメッセージに日本語で親しみやすく helpful に応答してください: %s", state.UserInput)
	
	// Use streaming for real-time updates
	var response strings.Builder
	_, err := llms.GenerateFromSinglePrompt(ctx, r.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		response.Write(chunk)
		state.OnStreamingChunk("handle_chat", string(chunk))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("failed to generate chat response: %w", err)
	}

	state.SetReport(response.String())
	return nil
}