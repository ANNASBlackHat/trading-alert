package mcp

import (
	"context"
	"path/filepath"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/joho/godotenv"
)

// ── Live-mongo tests (skip when MONGODB_URI is unreachable) ──

// loadEnvForTest loads the .env file at the repo root (parent of mcp/).
func loadEnvForTest(t *testing.T) {
	t.Helper()
	// Tests run with CWD = mcp/, so .env is one level up.
	if err := godotenv.Load(filepath.Join("..", ".env")); err != nil {
		t.Log("no .env at repo root; relying on environment")
	}
}

// initStoreForTest loads .env and connects to the real MONGODB_URI.
func initStoreForTest(t *testing.T) context.Context {
	t.Helper()
	loadEnvForTest(t)
	if err := Init(context.Background()); err != nil {
		t.Skipf("cannot connect to Mongo (%v); skipping", err)
	}
	if instance() == nil {
		t.Skip("store not initialized after Init; skipping")
	}
	return context.Background()
}

// TestBtcGetScoreboardLive exercises the scoreboard aggregation.
func TestBtcGetScoreboardLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := BtcGetScoreboard(ctx, &sdkmcp.CallToolRequest{}, BtcScoreboardArgs{Days: 30})
	if err != nil {
		t.Fatalf("BtcGetScoreboard: %v", err)
	}
	t.Logf("window=%d by_channel=%d by_confidence=%d by_timeframe=%d",
		out.WindowDays, len(out.ByChannel), len(out.ByConfidence), len(out.ByTimeframe))
}

// TestBtcGetTechniqueStatsLive exercises the technique ledger.
func TestBtcGetTechniqueStatsLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := BtcGetTechniqueStats(ctx, &sdkmcp.CallToolRequest{}, BtcTechniqueStatsArgs{})
	if err != nil {
		t.Fatalf("BtcGetTechniqueStats: %v", err)
	}
	t.Logf("techniques=%d", len(out.Techniques))
}

// TestBtcSearchAnalysesLive exercises the analyses search.
func TestBtcSearchAnalysesLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := BtcSearchAnalyses(ctx, &sdkmcp.CallToolRequest{}, BtcSearchAnalysesArgs{SinceDays: 7, Limit: 5})
	if err != nil {
		t.Fatalf("BtcSearchAnalyses: %v", err)
	}
	t.Logf("analyses=%d", out.Total)
}

// TestStocksTrendingLive exercises the trending-tickers aggregation.
func TestStocksTrendingLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := StocksTrendingHandler(ctx, &sdkmcp.CallToolRequest{}, StocksTrendingArgs{SinceDays: 30, TopN: 5})
	if err != nil {
		t.Fatalf("StocksTrending: %v", err)
	}
	t.Logf("tickers=%d", len(out.Tickers))
	for _, tr := range out.Tickers {
		t.Logf("  %s mentions=%d stance=%+v", tr.Ticker, tr.Mentions, tr.Stance)
	}
}

// TestStocksUpcomingCatalystsLive exercises the catalysts flattening.
func TestStocksUpcomingCatalystsLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := StocksUpcomingCatalystsHandler(ctx, &sdkmcp.CallToolRequest{}, StocksUpcomingCatalystsArgs{SinceDays: 30})
	if err != nil {
		t.Fatalf("StocksUpcomingCatalysts: %v", err)
	}
	t.Logf("catalysts=%d", len(out.Items))
	n := 5
	if len(out.Items) < n {
		n = len(out.Items)
	}
	for _, c := range out.Items[:n] {
		t.Logf("  %s / %s: %s", c.Ticker, c.CompanyName, c.Catalyst)
	}
}
