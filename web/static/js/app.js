// Main application logic for Go Research Assistant

class ResearchApp {
    constructor() {
        this.wsManager = new WebSocketManager();
        this.graphManager = new GraphManager();
        this.startTime = null;
        this.executionTimer = null;
        this.reportBuffer = '';
        this.currentReportNode = null;
        
        this.init();
    }

    init() {
        // Setup WebSocket event handling
        this.wsManager.onGraphUpdate((update) => this.handleGraphUpdate(update));
        
        // Connect WebSocket
        this.wsManager.connect();
        
        // Setup UI event handlers
        this.setupEventHandlers();
        
        // Initialize Showdown for markdown parsing
        this.initMarkdown();
    }

    setupEventHandlers() {
        // Start research button
        document.getElementById('startBtn').addEventListener('click', () => {
            this.startResearch();
        });

        // Enter key support for input
        document.getElementById('queryInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.startResearch();
            }
        });
    }

    initMarkdown() {
        // Debug: Check if Showdown.js is loaded
        if (typeof showdown === 'undefined') {
            console.error('Showdown.js is not loaded');
            this.addLog('⚠️ マークダウンライブラリが読み込まれていません', 'warning');
        } else {
            console.log('Showdown.js loaded successfully');
        }
    }

    handleGraphUpdate(update) {
        const { type, node, chunk, error } = update;
        
        switch(type) {
            case 'start':
                this.startExecution();
                break;
            case 'node_start':
                this.handleNodeStart(node);
                break;
            case 'node_complete':
                this.handleNodeComplete(node, update);
                break;
            case 'streaming_chunk':
                this.handleStreamingChunk(node, chunk);
                break;
            case 'complete':
                this.completeExecution();
                break;
            case 'error':
                this.handleError(node, error);
                break;
        }
    }

    handleNodeStart(node) {
        // Check if this is a dynamic search query node
        if (node.startsWith('search_query_')) {
            this.graphManager.createDynamicSearchNode(node);
            this.graphManager.updateNodeStatus(node, 'in-progress', '実行中...');
            this.addLog(`🔍 ${this.graphManager.getNodeDisplayName(node)}: 並行検索開始`, 'info');
        } else {
            this.graphManager.updateNodeStatus(node, 'in-progress', '実行中...');
            this.addLog(`📍 ${this.graphManager.getNodeDisplayName(node)}: 開始`, 'info');
        }
        
        // Start report section for report generation
        if (node === 'synthesize_and_report') {
            this.showReportSection();
            this.reportBuffer = '';
            this.currentReportNode = node;
        }
    }

    handleNodeComplete(node, update) {
        this.graphManager.updateNodeStatus(node, 'completed', '完了');
        
        // Special handling for different node types
        if (node === 'generate_search_queries') {
            const queryCount = update.state?.search_queries?.length || 0;
            this.addLog(`✅ ${this.graphManager.getNodeDisplayName(node)}: ${queryCount}個のクエリを生成`, 'success');
        } else if (node === 'merge_search_results') {
            const dynamicNodeCount = this.graphManager.getDynamicNodeCount();
            this.addLog(`✅ ${this.graphManager.getNodeDisplayName(node)}: ${dynamicNodeCount}個の検索結果を統合`, 'success');
        } else if (node.startsWith('search_query_')) {
            this.addLog(`✅ ${this.graphManager.getNodeDisplayName(node)}: 検索完了`, 'success');
        } else {
            this.addLog(`✅ ${this.graphManager.getNodeDisplayName(node)}: 完了`, 'success');
        }
        
        this.graphManager.updateProgress();
        
        // Finalize report if this was report generation
        if (node === 'synthesize_and_report') {
            this.currentReportNode = null;
            this.updateReportContent(this.reportBuffer); // Final update without progress indicator
        }
    }

    handleStreamingChunk(node, chunk) {
        if (chunk !== undefined && chunk !== null) {
            if (node === 'synthesize_and_report') {
                // Debug: Log each chunk to see what we're receiving
                console.log('Received chunk:', JSON.stringify(chunk));
                
                // Accumulate report content (don't trim to preserve newlines)
                this.reportBuffer += chunk;
                this.updateReportContent(this.reportBuffer);
            } else {
                // Regular streaming for other nodes (only show non-empty chunks)
                if (chunk.trim()) {
                    this.addStreamingChunk(node, chunk);
                }
            }
        }
    }

    handleError(node, error) {
        this.graphManager.updateNodeStatus(node, 'error', 'エラー');
        this.addLog(`❌ ${this.graphManager.getNodeDisplayName(node)}: ${error}`, 'error');
        this.wsManager.setResearchState(false);
    }

    startExecution() {
        this.startTime = Date.now();
        this.executionTimer = setInterval(() => this.updateExecutionTime(), 1000);
        document.getElementById('startBtn').disabled = true;
        this.graphManager.resetNodes();
        this.resetReportSection();
        this.addLog('🚀 調査を開始しました', 'info');
        this.wsManager.setResearchState(true, false);
    }

    completeExecution() {
        clearInterval(this.executionTimer);
        document.getElementById('startBtn').disabled = false;
        this.addLog('🎉 調査が完了しました', 'success');
        this.wsManager.setResearchState(false, true);
    }

    updateExecutionTime() {
        if (!this.startTime) return;
        const elapsed = Math.floor((Date.now() - this.startTime) / 1000);
        const minutes = Math.floor(elapsed / 60).toString().padStart(2, '0');
        const seconds = (elapsed % 60).toString().padStart(2, '0');
        document.getElementById('executionTime').textContent = `${minutes}:${seconds}`;
    }

    startResearch() {
        const query = document.getElementById('queryInput').value.trim();
        if (!query) {
            alert('調査したいトピックを入力してください');
            return;
        }
        
        if (this.wsManager.sendResearchRequest(query)) {
            document.getElementById('queryInput').value = '';
        }
    }

    addLog(message, type = 'info') {
        this.wsManager.addLog(message, type);
    }

    addStreamingChunk(node, chunk) {
        const output = document.getElementById('liveOutput');
        const entry = document.createElement('div');
        entry.className = 'log-entry';
        entry.innerHTML = `<span class="streaming-text">${this.graphManager.getNodeDisplayName(node)}</span>: ${chunk}`;
        output.appendChild(entry);
        output.scrollTop = output.scrollHeight;
    }

    // Report section functions
    showReportSection() {
        document.getElementById('reportContainer').style.display = 'block';
        const reportContent = document.getElementById('reportStreaming');
        reportContent.innerHTML = '<div class="report-progress">📝 レポートを生成中...</div>';
    }

    resetReportSection() {
        document.getElementById('reportContainer').style.display = 'none';
        document.getElementById('reportStreaming').innerHTML = '';
        this.reportBuffer = '';
        this.currentReportNode = null;
    }

    updateReportContent(content) {
        const reportElement = document.getElementById('reportStreaming');
        
        // Clean up content - only remove code blocks, don't mess with structure
        let cleanContent = content
            .replace(/```markdown\n?/g, '')
            .replace(/```\n?/g, '');
        
        // Debug: Show raw and cleaned markdown content
        console.log('Raw markdown content:', content);
        console.log('Cleaned markdown content (no regex fixes):', cleanContent);
        
        // Parse markdown to HTML using Showdown.js
        let formattedContent;
        try {
            if (typeof showdown !== 'undefined') {
                const converter = new showdown.Converter({
                    headerLevelStart: 1,
                    simplifiedAutoLink: true,
                    excludeTrailingPunctuationFromURLs: true,
                    literalMidWordUnderscores: true,
                    strikethrough: true,
                    tables: true,
                    tasklists: true,
                    smoothLivePreview: true,
                    smartIndentationFix: true
                });
                formattedContent = converter.makeHtml(cleanContent);
                console.log('Converted HTML:', formattedContent);
            } else {
                throw new Error('Showdown.js not loaded');
            }
        } catch (error) {
            console.error('Markdown parsing error:', error);
            // Fallback: Simple text with basic formatting
            formattedContent = cleanContent
                .replace(/^# (.*)/gm, '<h1>$1</h1>')
                .replace(/^## (.*)/gm, '<h2>$1</h2>')
                .replace(/^### (.*)/gm, '<h3>$1</h3>')
                .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
                .replace(/\n/g, '<br>');
        }
        
        // Add progress indicator if still streaming
        if (this.currentReportNode === 'synthesize_and_report') {
            formattedContent += '<div class="report-progress">✍️ 生成中...</div>';
        }
        
        reportElement.innerHTML = formattedContent;
        
        // Auto-scroll to bottom
        const reportContainer = document.getElementById('reportContent');
        reportContainer.scrollTop = reportContainer.scrollHeight;
    }
}

// Initialize the application when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.researchApp = new ResearchApp();
});