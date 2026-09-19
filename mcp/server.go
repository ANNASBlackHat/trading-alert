package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds the MCP server with all phase-1 tools, resources, and
// prompts registered. The caller is responsible for calling Init first
// (or ensuring MONGODB_URI is set and calling Init from cmd/mcp/main.go).
func NewServer() *mcp.Server {
	impl := &mcp.Implementation{
		Name:        "trading-agent-memory",
		Title:       "Trading Agent Memory",
		Description: "Read-only MCP server over the btc_agent and stocks_agent MongoDB databases, with live price context. Outputs are third-party analyst opinions and historical scores, not financial advice.",
		Version:     "0.1.0",
	}
	srv := mcp.NewServer(impl, nil)

	registerTools(srv)
	RegisterResources(srv)
	RegisterPrompts(srv)

	return srv
}

func registerTools(srv *mcp.Server) {
	// BTC
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "btc_get_agent_state",
		Description: "Return the btc_agent's rolling memory (narrative, consensus key levels, open predictions, current view). Set include_prices=true to attach the live BTC price. " + disclaimerText,
	}, BtcGetAgentState)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "btc_list_predictions",
		Description: "List BTC predictions, filtered by status (pending/scored/all), days lookback, and confidence. Unscored items are flagged. " + disclaimerText,
	}, BtcListPredictions)

	// Stocks
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_get_agent_state",
		Description: "Return the stocks_agent's rolling memory (narrative, sectors consensus, open predictions, current view). " + disclaimerText,
	}, StocksGetAgentState)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_get_ticker_view",
		Description: "All stock cards for a ticker with a stance breakdown and unresolved-ticker count. " + disclaimerText,
	}, StocksGetTickerView)

	// Shared
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_pipeline_status",
		Description: "Ingestion health: videos processed/failed, per-channel breakdown, and unresolved tickers in the window. " + disclaimerText,
	}, GetPipelineStatus)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_video_detail",
		Description: "Everything extracted from one video across all collections (video, speakers, stock cards, concept cards). " + disclaimerText,
	}, GetVideoDetail)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_channels",
		Description: "The tracked YouTube channels reference table. " + disclaimerText,
	}, ListChannels)
}
