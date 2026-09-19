package mcp

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// TestInMemoryToolHandlers spins up an in-memory Mongo, seeds it with the
// fixtures the schema docs describe, and exercises every phase-1 tool handler
// in-process. This verifies query shapes, filters, and aggregation pipelines
// without needing the Atlas cluster's DNS (which may be unreachable from a
// given network).
//
// It uses the official embedded Mongo when available; otherwise it skips.
func TestInMemoryToolHandlers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	srv, err := startInMemoryMongo(ctx)
	if err != nil {
		t.Skipf("in-memory mongo unavailable: %v", err)
	}
	defer srv.Stop()

	if err := Init(ctx); err != nil {
		t.Fatalf("Init against in-memory mongo: %v", err)
	}
	s := instance()
	seed(t, ctx, s)

	// ── btc_get_agent_state ───────────────────────────────
	if _, out, err := BtcGetAgentState(ctx, nil, BtcAgentStateArgs{}); err != nil {
		t.Fatalf("BtcGetAgentState: %v", err)
	} else if !out.Found {
		t.Error("BtcGetAgentState: expected found=true")
	}

	// ── btc_list_predictions ──────────────────────────────
	if _, out, err := BtcListPredictions(ctx, nil, BtcListPredictionsArgs{Status: "pending", Days: 30}); err != nil {
		t.Fatalf("BtcListPredictions: %v", err)
	} else if out.Total != 1 {
		t.Errorf("BtcListPredictions pending: got %d, want 1", out.Total)
	}

	// ── stocks_get_agent_state ────────────────────────────
	if _, out, err := StocksGetAgentState(ctx, nil, StocksAgentStateArgs{}); err != nil {
		t.Fatalf("StocksGetAgentState: %v", err)
	} else if !out.Found {
		t.Error("StocksGetAgentState: expected found=true")
	}

	// ── stocks_get_ticker_view ────────────────────────────
	if _, out, err := StocksGetTickerView(ctx, nil, StocksGetTickerViewArgs{Ticker: "NVDA", SinceDays: 30}); err != nil {
		t.Fatalf("StocksGetTickerView: %v", err)
	} else if len(out.Cards) != 2 {
		t.Errorf("StocksGetTickerView: got %d cards, want 2", len(out.Cards))
	} else if out.StanceBreakdown.Bull != 1 || out.StanceBreakdown.Bear != 1 {
		t.Errorf("stance breakdown: %+v", out.StanceBreakdown)
	}

	// ── get_video_detail ──────────────────────────────────
	if _, out, err := GetVideoDetail(ctx, nil, VideoDetailArgs{VideoID: "vid-1"}); err != nil {
		t.Fatalf("GetVideoDetail: %v", err)
	} else if len(out.Speakers) != 1 || len(out.StockCards) != 1 || len(out.ConceptCards) != 1 {
		t.Errorf("GetVideoDetail: speakers=%d stock=%d concept=%d", len(out.Speakers), len(out.StockCards), len(out.ConceptCards))
	}

	// ── get_pipeline_status ───────────────────────────────
	if _, out, err := GetPipelineStatus(ctx, nil, PipelineStatusArgs{SinceDays: 30}); err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	} else if out.VideosProcessed != 1 {
		t.Errorf("GetPipelineStatus processed=%d, want 1", out.VideosProcessed)
	}

	// ── list_channels ─────────────────────────────────────
	if _, out, err := ListChannels(ctx, nil, ListChannelsArgs{}); err != nil {
		t.Fatalf("ListChannels: %v", err)
	} else if len(out.Channels) != 2 {
		t.Errorf("ListChannels: got %d, want 2", len(out.Channels))
	}
}

func seed(t *testing.T, ctx context.Context, s *Store) {
	t.Helper()

	// btc_agent
	if _, err := s.btc.Collection("agent_memory").InsertOne(ctx, bson.M{
		"last_updated":         "2026-09-19T00:00:00Z",
		"market_narrative":     "ranging near resistance",
		"agent_current_view":   "bullish bias above 108k",
		"key_levels_consensus": bson.M{"support": []float64{96000}, "resistance": []float64{112000}},
		"open_predictions":     []bson.M{{"target": 112000.0, "target_date": "2026-09-26"}},
	}); err != nil {
		t.Fatalf("seed btc agent_memory: %v", err)
	}
	if _, err := s.btc.Collection("predictions").InsertOne(ctx, bson.M{
		"video_id":        "v1",
		"channel_name":    "Channel A",
		"prediction_date": "2026-09-01",
		"direction":       "up",
		"target":          112000.0,
		"timeframe":       "14 days",
		"target_date":     "2026-09-15",
		"confidence":      "high",
		"outcome":         nil,
	}); err != nil {
		t.Fatalf("seed predictions: %v", err)
	}

	// stocks_agent
	if _, err := s.stocks.Collection("agent_memory").InsertOne(ctx, bson.M{
		"last_updated":     "2026-09-19T00:00:00Z",
		"market_narrative": "risk-on, semis leading",
	}); err != nil {
		t.Fatalf("seed stocks agent_memory: %v", err)
	}

	if _, err := s.stocks.Collection("processed_videos").InsertOne(ctx, bson.M{
		"video_id": "vid-1", "schema_version": "v2", "channel_id": "ch-1",
		"publish_date": "2026-09-10", "status": "success",
	}); err != nil {
		t.Fatalf("seed processed_videos: %v", err)
	}
	for i, stance := range []string{"bull", "bear"} {
		if _, err := s.stocks.Collection("stock_cards").InsertOne(ctx, bson.M{
			"card_id": "c" + string(rune('A'+i)), "video_id": "vid-1", "schema_version": "v2",
			"ticker": "NVDA", "company_name": "NVIDIA", "sector": "Semiconductors",
			"stance": stance, "direct_quote": "NVDA looks strong", "confidence_language": "strong",
			"timeframe": "3-6 months",
		}); err != nil {
			t.Fatalf("seed stock_cards: %v", err)
		}
	}
	if _, err := s.stocks.Collection("speakers").InsertOne(ctx, bson.M{
		"video_id": "vid-1", "name": "Jane", "role": "analyst",
	}); err != nil {
		t.Fatalf("seed speakers: %v", err)
	}
	if _, err := s.stocks.Collection("concept_cards").InsertOne(ctx, bson.M{
		"card_id": "k1", "video_id": "vid-1", "schema_version": "v2",
		"concept": "AI capex cycle", "concept_category": "macro",
		"explanation_summary": "spending on datacenters",
	}); err != nil {
		t.Fatalf("seed concept_cards: %v", err)
	}
	for _, cid := range []string{"ch-1", "ch-2"} {
		if _, err := s.stocks.Collection("channels").InsertOne(ctx, bson.M{
			"channel_id": cid, "name": "Name " + cid,
		}); err != nil {
			t.Fatalf("seed channels: %v", err)
		}
	}
}
