// Graph visualization and dynamic node management

class GraphManager {
    constructor() {
        this.nodeMapping = {
            'classify_intent_and_topic': 'node-classify',
            'generate_search_queries': 'node-queries', 
            'execute_parallel_search': 'node-search',
            'merge_search_results': 'node-merge',
            'synthesize_and_report': 'node-report'
        };
        
        this.dynamicNodes = [];
        this.originalFlow = null;
    }

    updateNodeStatus(nodeName, status, statusText) {
        let nodeId = this.nodeMapping[nodeName];
        
        // If not in static mapping, check if it's a dynamic node
        if (!nodeId && nodeName.startsWith('search_query_')) {
            nodeId = 'dynamic-' + nodeName;
        }
        
        if (!nodeId) {
            console.log(`Node ID not found for: ${nodeName}`);
            return;
        }
        
        const node = document.getElementById(nodeId);
        if (!node) {
            console.log(`Node element not found for ID: ${nodeId}`);
            return;
        }
        
        node.className = `node ${status}`;
        const statusElement = node.querySelector('.node-status');
        if (statusElement) {
            statusElement.textContent = statusText;
        }
    }

    getNodeDisplayName(nodeName) {
        const names = {
            'classify_intent_and_topic': '意図判定',
            'generate_search_queries': 'クエリ生成',
            'execute_parallel_search': '並行検索',
            'merge_search_results': '結果合流',
            'synthesize_and_report': 'レポート生成'
        };
        
        // Handle dynamic search query nodes
        if (nodeName.startsWith('search_query_')) {
            const queryNum = nodeName.replace('search_query_', '');
            return `検索${queryNum}`;
        }
        
        return names[nodeName] || nodeName;
    }

    createDynamicSearchNode(nodeName) {
        // Check if node already exists
        const nodeId = 'dynamic-' + nodeName;
        if (document.getElementById(nodeId)) {
            return; // Node already exists
        }
        
        // Get the graph flow container
        const graphFlow = document.querySelector('.graph-flow');
        
        // Save original flow if not already saved
        if (!this.originalFlow) {
            this.originalFlow = graphFlow.innerHTML;
        }
        
        // Check if we need to create the dynamic branching layout
        if (this.dynamicNodes.length === 0) {
            this.createDynamicBranchingLayout();
        }
        
        // Create the dynamic node
        const node = document.createElement('div');
        node.className = 'node pending';
        node.id = nodeId;
        node.innerHTML = `
            <div class="node-title">🔍 ${this.getNodeDisplayName(nodeName)}</div>
            <div class="node-status">待機中</div>
        `;
        
        // Add to dynamic nodes list
        this.dynamicNodes.push(nodeName);
        
        // Insert into the branching area
        const branchingArea = document.getElementById('dynamic-branching-area');
        if (branchingArea) {
            // Update branch info with current count
            const branchInfo = branchingArea.querySelector('.branch-info');
            if (branchInfo) {
                branchInfo.textContent = `${this.dynamicNodes.length}個の並行検索実行中`;
            }
            
            branchingArea.appendChild(node);
        }
        
        console.log(`Created dynamic node: ${nodeName} (total: ${this.dynamicNodes.length})`);
    }

    createDynamicBranchingLayout() {
        const graphFlow = document.querySelector('.graph-flow');
        
        // Create new dynamic layout
        graphFlow.innerHTML = `
            <div class="node pending" id="node-classify">
                <div class="node-title">🎯 意図判定</div>
                <div class="node-status">待機中</div>
            </div>
            <div class="arrow">→</div>
            <div class="node pending" id="node-queries">
                <div class="node-title">🔍 クエリ生成</div>
                <div class="node-status">待機中</div>
            </div>
            <div class="arrow">↓</div>
            <div class="branching-container">
                <div class="node pending" id="node-merge">
                    <div class="node-title">🔗 結果合流</div>
                    <div class="node-status">待機中</div>
                </div>
                <div class="dynamic-search-area" id="dynamic-branching-area">
                    <div class="branch-info">検索クエリの並行実行</div>
                    <!-- Dynamic search nodes will be inserted here -->
                </div>
            </div>
            <div class="arrow">→</div>
            <div class="node pending" id="node-report">
                <div class="node-title">📝 レポート生成</div>
                <div class="node-status">待機中</div>
            </div>
        `;
    }

    resetNodes() {
        // Reset dynamic nodes and layout
        this.dynamicNodes = [];
        
        // Restore original layout if it was changed
        if (this.originalFlow) {
            const graphFlow = document.querySelector('.graph-flow');
            graphFlow.innerHTML = this.originalFlow;
            this.originalFlow = null;
        }
        
        // Reset all static nodes
        Object.values(this.nodeMapping).forEach(nodeId => {
            const node = document.getElementById(nodeId);
            if (node) {
                node.className = 'node pending';
                const statusElement = node.querySelector('.node-status');
                if (statusElement) {
                    statusElement.textContent = '待機中';
                }
            }
        });
        
        // Update progress with actual node count
        setTimeout(() => {
            const total = document.querySelectorAll('.node').length;
            document.getElementById('progress').textContent = `0/${total}`;
        }, 100);
    }

    updateProgress() {
        const completed = document.querySelectorAll('.node.completed').length;
        const total = document.querySelectorAll('.node').length;
        document.getElementById('progress').textContent = `${completed}/${total}`;
    }

    getDynamicNodeCount() {
        return this.dynamicNodes.length;
    }
}

// Export for use in other modules
window.GraphManager = GraphManager;