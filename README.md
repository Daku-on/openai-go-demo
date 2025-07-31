# 🚀 Go フルスタック リサーチアシスタント

**純粋なGoで構築された**マルチプラットフォーム対応の強力な自律型調査アシスタント！グラフベース処理、リアルタイムストリーミング、並列検索による包括的な調査機能を提供します。

## 🎯 3つのインターフェース

| インターフェース | 説明 | 適用場面 |
|------------------|------|----------|
| 🖥️ **CLI版** | ターミナルベースのインターフェース | パワーユーザー、スクリプト、サーバー環境 |
| 🌐 **Web版** | リアルタイムグラフ可視化 | インタラクティブ分析、チーム共同作業 |  
| ⚡ **WASM版** | ブラウザネイティブGo実行 | オフライン利用、クライアントサイド処理 |

## ✨ 主な特徴

- **意図ベースルーティング**: ユーザー入力を自動的に調査、Q&A、雑談に分類
- **自律的調査**: 複数の検索クエリを生成し並列実行
- **包括的レポート**: 検索結果を構造化されたレポートに統合
- **ストリーミング更新**: グラフ実行中のリアルタイム進捗表示
- **並行処理**: Goのgoroutineを活用した効率的な並列検索
- **実際のWeb検索**: SerpAPIによる本物のGoogle検索結果
- **ノード状態可視化**: Web版でのリアルタイム実行状態表示

## 🏗️ アーキテクチャ

LangGraphにインスパイアされたグラフベースアーキテクチャ：

1. **状態管理**: スレッドセーフな`AppState`が実行全体のコンテキストを維持
2. **ノード**: 個別の処理ユニット（意図分類、クエリ生成、検索、統合）
3. **エッジ**: 状態に基づくノード遷移の制御フロー
4. **エンジン**: ストリーミング更新をサポートするグラフ実行オーケストレーション

### グラフフロー

```
ユーザー入力 → 意図分類 → [意図に基づく分岐]
                          ├─ 調査 → クエリ生成 → 並列検索 → レポート統合
                          ├─ Q&A → 直接回答
                          └─ 雑談 → チャット処理
```

## 📋 必要条件

- Go 1.21以上
- OpenAI APIキー
- SerpAPIキー（オプション、実際のWeb検索用）

## 🚀 インストール

1. リポジトリをクローン:
```bash
git clone https://github.com/takako/openai-go-demo.git
cd openai-go-demo
```

2. 依存関係をインストール:
```bash
make deps
```

3. 環境変数ファイルを作成:
```bash
cp .env.example .env
```

4. APIキーを`.env`に追加:
```
OPENAI_API_KEY=your-openai-api-key-here
SERPAPI_KEY=your-serpapi-key-here
```

**APIキーの取得:**
- OpenAI API: https://platform.openai.com/api-keys
- SerpAPI（実際のGoogle検索用）: https://serpapi.com/manage-api-key

## 🎮 使用方法

### すべてのバージョンをビルド
```bash
make build-all
```

### 1. CLI版の実行
```bash
make build-cli
./bin/research-cli
```

### 2. Web版の実行（推奨！）
```bash
make serve-web
# http://localhost:8080 をブラウザで開く
```

**Web版の特徴:**
- **リアルタイムノード状態可視化**
- **パルス効果付きグラフ表示**
- **WebSocketストリーミング**
- **LIVE出力表示**

### 3. WASM版の実行
```bash
make serve-wasm
# http://localhost:3000 をブラウザで開く
```

**WASM版の特徴:**
- **ブラウザ内で完全Go実行**
- **オフライン動作可能**
- **クライアントサイド処理**

### インタラクティブコマンド

- 調査したいトピックを入力
- `stream` - ストリーミングモードの切り替え
- `exit` または `quit` - アプリケーション終了

### クエリの例

**調査モード:**
- "Go言語の最新動向について教えて"
- "量子コンピュータの現在の開発状況を調べて"
- "AI規制の最新状況について調査して"

**Q&Aモード:**
- "フランスの首都は？"
- "goroutineの仕組みは？"

**チャットモード:**
- "こんにちは"
- "ありがとう"

## 📁 プロジェクト構成

```
📁 openai-go-demo/
├── 🖥️  cmd/cli/main.go     ← CLI版メイン
├── 🌐 cmd/web/main.go     ← WebSocketサーバー
├── ⚡ cmd/wasm/main.go    ← WASM版メイン
├── 🧠 graph/              ← 共通ロジック
│   ├── state.go          ← AppState定義と管理
│   ├── nodes.go          ← ノード実装
│   ├── edges.go          ← エッジロジック
│   └── engine.go         ← グラフ実行エンジン
├── 🔍 tools/serpapi.go    ← SerpAPI検索クライアント
├── 🎨 web/static/         ← Web UI（グラフ可視化）
├── ⚡ wasm/               ← WASM UI（ブラウザネイティブ）
├── 📋 Makefile           ← ビルドスクリプト
└── 📄 .env               ← 環境変数（リポジトリに含まず）
```

## 🔧 主要コンポーネント

### AppState
スレッドセーフな操作によるノード間共有状態管理:
- ユーザー入力と意図分類
- 検索クエリと結果
- 最終レポート生成
- エラーハンドリング

### ノード
- **ClassifyIntentAndTopic**: ユーザー意図の判定と調査トピック抽出
- **GenerateSearchQueries**: 包括的調査のための多様な検索クエリ生成
- **ExecuteParallelSearch**: goroutineを使用した並行検索実行
- **SynthesizeAndReport**: 結果を構造化レポートに統合

### エンジン
以下の機能でグラフ実行をオーケストレーション:
- 無限ループ防止のための最大ステップ制限
- リアルタイム進捗のストリーミング更新
- 実行パス追跡
- エラー伝播とハンドリング

## 🛠️ Makefileコマンド

```bash
make help          # 利用可能なコマンド一覧
make build-all     # 全バージョンビルド
make build-cli     # CLI版ビルド
make build-web     # Web版ビルド
make build-wasm    # WASM版ビルド
make serve-web     # Web版サーバー起動
make serve-wasm    # WASM版サーバー起動
make demo          # デモ環境セットアップ
make clean         # ビルド成果物削除
make test          # テスト実行
```

## 🎨 拡張方法

### 新しいノードの追加

1. `nodes.go`でノード関数を定義:
```go
func (r *NodeRegistry) YourNewNode(ctx context.Context, state *AppState) error {
    // ノードロジック
    return nil
}
```

2. `NewNodeRegistry`でノードを登録:
```go
registry.RegisterNode("your_new_node", registry.YourNewNode)
```

3. `edges.go`でエッジロジックを更新

### 実際の検索ツールの追加

`ExecuteParallelSearch`の模擬検索を実際の検索APIに置換:
- Tavily API（Web検索）
- GitHub API（コード検索）
- arXiv API（学術論文検索）

## ⚡ パフォーマンス考慮事項

- **並列実行**: 検索操作が並行実行され、総実行時間を大幅短縮
- **ストリーミング更新**: 進捗更新がメイン実行をブロックしない
- **スレッドセーフ状態**: 並行操作が共有状態を安全に更新
- **コンテキストキャンセレーション**: 適切なクリーンアップのための全操作でのコンテキストサポート

## 🔒 ライセンス

このプロジェクトは実演目的です。OpenAIの利用規約を遵守してください。

---

# 🚀 Go Full-Stack Research Assistant

A powerful autonomous research assistant built with **pure Go** across multiple platforms! Features graph-based processing, real-time streaming, and comprehensive research using parallel search operations.

## 🎯 Multiple Interfaces Available

| Interface | Description | Best For |
|-----------|-------------|----------|
| 🖥️ **CLI** | Terminal-based interface | Power users, scripting, server environments |
| 🌐 **Web** | Real-time graph visualizer | Interactive analysis, team collaboration |  
| ⚡ **WASM** | Browser-native Go execution | Offline use, client-side processing |

## ✨ Features

- **Intent-based Routing**: Automatically classifies user input as research requests, Q&A, or general chat
- **Autonomous Research**: Generates multiple search queries and executes them in parallel
- **Comprehensive Reports**: Synthesizes search results into well-structured research reports
- **Streaming Updates**: Real-time progress updates during graph execution
- **Concurrent Processing**: Leverages Go's goroutines for efficient parallel search operations
- **Real Web Search**: Actual Google search results via SerpAPI
- **Node State Visualization**: Real-time execution state display in Web version

## 🏗️ Architecture

The application uses a graph-based architecture inspired by LangGraph:

1. **State Management**: Thread-safe `AppState` maintains context throughout execution
2. **Nodes**: Individual processing units (intent classification, query generation, search, synthesis)
3. **Edges**: Control flow logic that determines node transitions based on state
4. **Engine**: Orchestrates graph execution with support for streaming updates

### Graph Flow

```
User Input → Classify Intent → [Based on Intent]
                                ├─ Research → Generate Queries → Parallel Search → Synthesize Report
                                ├─ Q&A → Answer Directly
                                └─ Chat → Handle Chat
```

## 📋 Prerequisites

- Go 1.21 or higher
- OpenAI API key
- SerpAPI key (optional, for real web search)

## 🚀 Installation

1. Clone the repository:
```bash
git clone https://github.com/takako/openai-go-demo.git
cd openai-go-demo
```

2. Install dependencies:
```bash
make deps
```

3. Create a `.env` file:
```bash
cp .env.example .env
```

4. Add your API keys to `.env`:
```
OPENAI_API_KEY=your-openai-api-key-here
SERPAPI_KEY=your-serpapi-key-here
```

**Getting API Keys:**
- OpenAI API: https://platform.openai.com/api-keys
- SerpAPI (for real web search): https://serpapi.com/manage-api-key

## 🎮 Usage

### Build All Versions
```bash
make build-all
```

### 1. CLI Version
```bash
make build-cli
./bin/research-cli
```

### 2. Web Version (Recommended!)
```bash
make serve-web
# Open http://localhost:8080 in browser
```

**Web Version Features:**
- **Real-time node state visualization**
- **Pulsing graph display effects**
- **WebSocket streaming**
- **LIVE output display**

### 3. WASM Version
```bash
make serve-wasm
# Open http://localhost:3000 in browser
```

**WASM Version Features:**
- **Complete Go execution in browser**
- **Offline operation capability**
- **Client-side processing**

### Interactive Commands

- Type your query or research topic
- `stream` - Toggle streaming mode for real-time updates
- `exit` or `quit` - Exit the application

### Example Queries

**Research Mode:**
- "Research the latest developments in quantum computing"
- "Tell me about WebAssembly support in Go"
- "Investigate the current state of AI regulation"

**Q&A Mode:**
- "What is the capital of France?"
- "How do goroutines work?"

**Chat Mode:**
- "Hello, how are you?"
- "Thanks for your help!"

## 📁 Project Structure

```
📁 openai-go-demo/
├── 🖥️  cmd/cli/main.go     ← CLI version main
├── 🌐 cmd/web/main.go     ← WebSocket server
├── ⚡ cmd/wasm/main.go    ← WASM version main
├── 🧠 graph/              ← Common logic
│   ├── state.go          ← AppState definition and management
│   ├── nodes.go          ← Node implementations
│   ├── edges.go          ← Edge logic for transitions
│   └── engine.go         ← Graph execution engine
├── 🔍 tools/serpapi.go    ← SerpAPI search client
├── 🎨 web/static/         ← Web UI (graph visualization)
├── ⚡ wasm/               ← WASM UI (browser native)
├── 📋 Makefile           ← Build scripts
└── 📄 .env               ← Environment variables (not in repo)
```

## 🔧 Key Components

### AppState
Manages the shared state across all nodes with thread-safe operations:
- User input and intent classification
- Search queries and results
- Final report generation
- Error handling

### Nodes
- **ClassifyIntentAndTopic**: Determines user intent and extracts research topics
- **GenerateSearchQueries**: Creates diverse search queries for comprehensive research
- **ExecuteParallelSearch**: Performs concurrent searches using goroutines
- **SynthesizeAndReport**: Combines results into structured reports

### Engine
Orchestrates graph execution with features:
- Maximum step limit to prevent infinite loops
- Streaming updates for real-time progress
- Execution path tracking
- Error propagation and handling

## 🛠️ Makefile Commands

```bash
make help          # Show available commands
make build-all     # Build all versions
make build-cli     # Build CLI version
make build-web     # Build Web version
make build-wasm    # Build WASM version
make serve-web     # Start Web server
make serve-wasm    # Start WASM server
make demo          # Setup demo environment
make clean         # Clean build artifacts
make test          # Run tests
```

## 🎨 Extending the Application

### Adding New Nodes

1. Define the node function in `nodes.go`:
```go
func (r *NodeRegistry) YourNewNode(ctx context.Context, state *AppState) error {
    // Your node logic
    return nil
}
```

2. Register the node in `NewNodeRegistry`:
```go
registry.RegisterNode("your_new_node", registry.YourNewNode)
```

3. Update edge logic in `edges.go` to route to your node

### Adding Real Search Tools

Replace the simulated search in `ExecuteParallelSearch` with actual search APIs:
- Tavily API for web search
- GitHub API for code search
- arXiv API for academic papers

## ⚡ Performance Considerations

- **Parallel Execution**: Search operations run concurrently, significantly reducing total execution time
- **Streaming Updates**: Progress updates don't block main execution
- **Thread-Safe State**: Concurrent operations safely update shared state
- **Context Cancellation**: All operations support context for proper cleanup

## 🔒 License

This project is for demonstration purposes. Please ensure you comply with OpenAI's usage policies.