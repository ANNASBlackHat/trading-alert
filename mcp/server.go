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
		Version:     "0.3.0",
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

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "btc_get_scoreboard",
		Description: "Aggregate BTC prediction accuracy by channel, confidence level, and timeframe over a date window. Small-sample groups are flagged. " + disclaimerText,
	}, BtcGetScoreboard)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "btc_technique_stats",
		Description: "Technique ledger sorted by hit-rate, with times_used, correct_calls, best_market_condition, and a low-sample flag. " + disclaimerText,
	}, BtcGetTechniqueStats)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "btc_search_analyses",
		Description: "Search BTC daily analyses by date range, channel, market structure, or technique. Set include_transcript=true to include the raw transcription. " + disclaimerText,
	}, BtcSearchAnalyses)

	// Stocks
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_get_agent_state",
		Description: "Return the stocks_agent's rolling memory (narrative, sectors consensus, open predictions, current view). " + disclaimerText,
	}, StocksGetAgentState)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_get_ticker_view",
		Description: "All stock cards for a ticker with a stance breakdown and unresolved-ticker count. " + disclaimerText,
	}, StocksGetTickerView)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_trending",
		Description: "Most-mentioned tickers over a window, with mention count and bull/bear/neutral stance split. " + disclaimerText,
	}, StocksTrendingHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stocks_upcoming_catalysts",
		Description: "All upcoming catalysts mentioned in stock cards over a window, grouped by ticker with source video. " + disclaimerText,
	}, StocksUpcomingCatalystsHandler)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_stock_price_context",
		Description: "Live prices for arbitrary tickers (Finnhub) with the most-recent analyst price target attached. Works for any stock, not just the ones already mentioned. " + disclaimerText,
	}, GetStockPriceContext)

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
