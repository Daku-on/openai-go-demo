// WebSocket management for Go Research Assistant

class WebSocketManager {
    constructor() {
        this.ws = null;
        this.researchCompleted = false;
        this.isResearching = false;
        this.onGraphUpdateCallback = null;
    }

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        this.setupEventHandlers();
    }

    setupEventHandlers() {
        this.ws.onopen = () => {
            this.updateConnectionStatus(true);
            this.addLog('WebSocket接続が確立されました', 'success');
        };

        this.ws.onmessage = (event) => {
            const update = JSON.parse(event.data);
            if (this.onGraphUpdateCallback) {
                this.onGraphUpdateCallback(update);
            }
        };

        this.ws.onclose = (event) => {
            this.updateConnectionStatus(false);
            
            // Only show error if disconnection was unexpected
            if (!this.researchCompleted && this.isResearching) {
                this.addLog('⚠️ 接続が予期せず切断されました', 'warning');
                // Only auto-reconnect if research was in progress
                setTimeout(() => this.connect(), 3000);
            } else if (this.researchCompleted) {
                // Normal completion - no error message needed
            } else {
                // Initial connection or idle state
                this.addLog('接続が切断されました', 'info');
                setTimeout(() => this.connect(), 5000);
            }
        };

        this.ws.onerror = (error) => {
            if (this.isResearching) {
                this.addLog('⚠️ 接続エラーが発生しました', 'warning');
            }
        };
    }

    updateConnectionStatus(connected) {
        const status = document.getElementById('connectionStatus');
        const text = document.getElementById('connectionText');
        
        if (connected) {
            status.className = 'connection-status connected';
            text.textContent = '接続済み';
        } else {
            status.className = 'connection-status disconnected';
            text.textContent = '切断';
        }
    }

    sendResearchRequest(query) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'research',
                query: query
            }));
            return true;
        } else {
            this.addLog('WebSocket接続がありません', 'error');
            return false;
        }
    }

    setResearchState(researching, completed = false) {
        this.isResearching = researching;
        this.researchCompleted = completed;
    }

    onGraphUpdate(callback) {
        this.onGraphUpdateCallback = callback;
    }

    addLog(message, type = 'info') {
        const output = document.getElementById('liveOutput');
        const entry = document.createElement('div');
        entry.className = `log-entry ${type}`;
        entry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
        output.appendChild(entry);
        output.scrollTop = output.scrollHeight;
    }
}

// Export for use in other modules
window.WebSocketManager = WebSocketManager;