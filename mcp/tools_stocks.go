package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ── stocks_get_agent_state ───────────────────────────────────

// StocksAgentStateArgs has no required fields.
type StocksAgentStateArgs struct{}

// StocksAgentState is the response for stocks_get_agent_state.
type StocksAgentState struct {
	Found      bool               `json:"found"`
	Memory     *StocksAgentMemory `json:"memory,omitempty"`
	Unscored   int                `json:"unscored,omitempty"`
	Disclaimer string             `json:"disclaimer"`
}

// StocksGetAgentState returns the stocks_agent singleton memory.
func StocksGetAgentState(ctx context.Context, _ *sdkmcp.CallToolRequest, in StocksAgentStateArgs) (*sdkmcp.CallToolResult, StocksAgentState, error) {
	s := instance()
	if s == nil {
		return nil, StocksAgentState{}, storeErr()
	}
	var mem StocksAgentMemory
	err := s.stocks.Collection("agent_memory").
		FindOne(ctx, bson.M{}, options.FindOne()).Decode(&mem)

	out := StocksAgentState{Disclaimer: disclaimerText}
	if err == mongo.ErrNoDocuments {
		out.Found = false
		return nil, out, nil
	}
	if err != nil {
		return nil, out, fmt.Errorf("read stocks agent_memory: %w", err)
	}

	out.Found = true
	out.Memory = &mem
	for _, p := range mem.OpenPredictions {
		if outcome, _ := p["outcome"].(string); outcome == "" {
			out.Unscored++
		}
	}
	return nil, out, nil
}

// ── stocks_get_ticker_view ───────────────────────────────────

// StocksGetTickerViewArgs filters the stock_cards collection by ticker.
type StocksGetTickerViewArgs struct {
	Ticker        string `json:"ticker" jsonschema:"ticker symbol, e.g. NVDA"`
	SinceDays     int    `json:"since_days,omitempty" jsonschema:"only cards from N days back (default 30)"`
	IncludeQuotes bool   `json:"include_quotes,omitempty" jsonschema:"include direct_quote text (default true)"`
	Limit         int    `json:"limit,omitempty" jsonschema:"max cards (default 50, cap 200)"`
}

// StocksTickerView is the response for stocks_get_ticker_view.
type StocksTickerView struct {
	Ticker          string          `json:"ticker"`
	Cards           []StockCard     `json:"cards"`
	StanceBreakdown StanceBreakdown `json:"stance_breakdown"`
	UnresolvedCards int             `json:"data_quality.unresolved_ticker_cards,omitempty"`
	Disclaimer      string          `json:"disclaimer"`
}

// StocksGetTickerView returns all stock cards for a ticker with a stance
// breakdown and data-quality flag.
func StocksGetTickerView(ctx context.Context, _ *sdkmcp.CallToolRequest, in StocksGetTickerViewArgs) (*sdkmcp.CallToolResult, StocksTickerView, error) {
	s := instance()
	if s == nil {
		return nil, StocksTickerView{}, storeErr()
	}

	if in.SinceDays <= 0 {
		in.SinceDays = 30
	}
	if in.Limit <= 0 {
		in.Limit = 50
	}
	if in.Limit > 200 {
		in.Limit = 200
	}

	since := time.Now().UTC().AddDate(0, 0, -in.SinceDays).Format("2006-01-02")

	coll := s.stocks.Collection("stock_cards")
	ticker := strings.ToUpper(in.Ticker)

	// Match cards where the ticker field is set to this symbol, OR the
	// company name contains it. Join processed_videos to scope the window.
	pipe := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"$or": []bson.M{
			{"ticker": ticker},
			{"company_name": bson.M{"$regex": ticker, "$options": "i"}},
		}}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "processed_videos",
			"localField":   "video_id",
			"foreignField": "video_id",
			"as":           "video",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$video", "preserveNullAndEmptyArrays": true}}},
		bson.D{{Key: "$match", Value: bson.M{"video.publish_date": bson.M{"$gte": since}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "video.publish_date", Value: -1}}}},
		bson.D{{Key: "$limit", Value: int64(in.Limit)}},
	}

	cursor, err := coll.Aggregate(ctx, pipe)
	if err != nil {
		return nil, StocksTickerView{}, fmt.Errorf("query stock_cards: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode each document into a raw map, then hydrate the typed struct so
	// heterogeneous LLM fields (string vs array) never cause a whole-batch
	// decode failure.
	var raw []map[string]any
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, StocksTickerView{}, fmt.Errorf("decode stock_cards: %w", err)
	}

	cards := make([]StockCard, 0, len(raw))
	for _, m := range raw {
		cards = append(cards, hydrateStockCard(m))
	}

	out := StocksTickerView{
		Ticker:     ticker,
		Cards:      cards,
		Disclaimer: disclaimerText,
	}

	for i := range cards {
		c := &cards[i]
		switch strings.ToLower(c.Stance) {
		case "bull", "buy", "long", "positive":
			out.StanceBreakdown.Bull++
		case "bear", "sell", "short", "negative":
			out.StanceBreakdown.Bear++
		case "neutral", "mixed", "wait":
			out.StanceBreakdown.Neutral++
		default:
			out.StanceBreakdown.Other++
		}
		if c.TickerUnresolved {
			out.UnresolvedCards++
		}
	}
	if !in.IncludeQuotes {
		for i := range cards {
			cards[i].DirectQuote = ""
		}
		out.Cards = cards
	}

	return nil, out, nil
}
