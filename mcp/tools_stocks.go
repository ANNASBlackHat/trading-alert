package mcp

import (
	"context"
	"fmt"
	"sort"
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
		switch ClassifyStance(c.Stance) {
		case "bull":
			out.StanceBreakdown.Bull++
		case "bear":
			out.StanceBreakdown.Bear++
		case "neutral":
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

// ── stocks_trending ──────────────────────────────────────────

// StocksTrendingArgs filters the trending-tickers query.
type StocksTrendingArgs struct {
	SinceDays int `json:"since_days,omitempty" jsonschema:"window in days (default 14)"`
	TopN      int `json:"top_n,omitempty" jsonschema:"how many tickers to return (default 10, cap 50)"`
}

// StocksTrending returns the most-mentioned tickers in the window with a
// stance breakdown per ticker.
func StocksTrendingHandler(ctx context.Context, _ *sdkmcp.CallToolRequest, in StocksTrendingArgs) (*sdkmcp.CallToolResult, StocksTrendingOut, error) {
	s := instance()
	if s == nil {
		return nil, StocksTrendingOut{}, storeErr()
	}

	if in.SinceDays <= 0 {
		in.SinceDays = 14
	}
	if in.TopN <= 0 {
		in.TopN = 10
	}
	if in.TopN > 50 {
		in.TopN = 50
	}
	since := time.Now().UTC().AddDate(0, 0, -in.SinceDays).Format("2006-01-02")

	pipe := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"schema_version": "v2"}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "processed_videos",
			"localField":   "video_id",
			"foreignField": "video_id",
			"as":           "video",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$video", "preserveNullAndEmptyArrays": true}}},
		bson.D{{Key: "$match", Value: bson.M{"video.publish_date": bson.M{"$gte": since}}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$ticker",
			"cards": bson.M{"$push": bson.M{"stance": "$stance"}},
		}}},
	}

	cursor, err := s.stocks.Collection("stock_cards").Aggregate(ctx, pipe)
	if err != nil {
		return nil, StocksTrendingOut{}, fmt.Errorf("aggregate stock_cards: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []struct {
		Ticker *string          `bson:"_id"`
		Cards  []map[string]any `bson:"cards"`
	}
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, StocksTrendingOut{}, fmt.Errorf("decode trending groups: %w", err)
	}

	out := StocksTrendingOut{WindowDays: in.SinceDays, Disclaimer: disclaimerText}
	for _, g := range groups {
		if g.Ticker == nil || strings.TrimSpace(*g.Ticker) == "" {
			continue
		}
		t := TickerTrend{Ticker: strings.ToUpper(*g.Ticker)}
		for _, c := range g.Cards {
			stance := ClassifyStance(coerceStr(c["stance"]))
			t.Mentions++
			switch stance {
			case "bull":
				t.Stance.Bull++
			case "bear":
				t.Stance.Bear++
			case "neutral":
				t.Stance.Neutral++
			default:
				t.Stance.Other++
			}
		}
		out.Tickers = append(out.Tickers, t)
	}

	sort.Slice(out.Tickers, func(i, j int) bool {
		return out.Tickers[i].Mentions > out.Tickers[j].Mentions
	})
	if len(out.Tickers) > in.TopN {
		out.Tickers = out.Tickers[:in.TopN]
	}
	return nil, out, nil
}

// ── stocks_upcoming_catalysts ─────────────────────────────────

// StocksUpcomingCatalystsArgs has one optional field.
type StocksUpcomingCatalystsArgs struct {
	SinceDays int `json:"since_days,omitempty" jsonschema:"window in days (default 30)"`
}

// StocksUpcomingCatalysts returns all upcoming catalysts mentioned in stock
// cards in the window, with their source card / ticker / company.
func StocksUpcomingCatalystsHandler(ctx context.Context, _ *sdkmcp.CallToolRequest, in StocksUpcomingCatalystsArgs) (*sdkmcp.CallToolResult, StocksUpcomingCatalystsOut, error) {
	s := instance()
	if s == nil {
		return nil, StocksUpcomingCatalystsOut{}, storeErr()
	}
	if in.SinceDays <= 0 {
		in.SinceDays = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -in.SinceDays).Format("2006-01-02")

	// Fetch cards in the window (ticker, company, card_id, video_id,
	// upcoming_catalysts). Use $match + $lookup + $unwind.
	pipe := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"schema_version": "v2"}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "processed_videos",
			"localField":   "video_id",
			"foreignField": "video_id",
			"as":           "video",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$video", "preserveNullAndEmptyArrays": true}}},
		bson.D{{Key: "$match", Value: bson.M{"video.publish_date": bson.M{"$gte": since}}}},
		bson.D{{Key: "$project", Value: bson.M{
			"card_id":            true,
			"video_id":           true,
			"ticker":             true,
			"company_name":       true,
			"upcoming_catalysts": true,
		}}},
	}

	cursor, err := s.stocks.Collection("stock_cards").Aggregate(ctx, pipe)
	if err != nil {
		return nil, StocksUpcomingCatalystsOut{}, fmt.Errorf("aggregate catalysts: %w", err)
	}
	defer cursor.Close(ctx)

	var raw []map[string]any
	if err := cursor.All(ctx, &raw); err != nil {
		return nil, StocksUpcomingCatalystsOut{}, fmt.Errorf("decode catalysts: %w", err)
	}

	out := StocksUpcomingCatalystsOut{WindowDays: in.SinceDays, Disclaimer: disclaimerText}
	seen := map[string]bool{}
	for _, m := range raw {
		cardID := coerceStr(m["card_id"])
		company := coerceStr(m["company_name"])
		ticker := coerceStr(m["ticker"])
		videoID := coerceStr(m["video_id"])
		for _, cat := range toStrList(m["upcoming_catalysts"]) {
			if cat == "" {
				continue
			}
			dedup := cardID + "|" + cat
			if seen[dedup] {
				continue
			}
			seen[dedup] = true
			out.Items = append(out.Items, CatalystItem{
				Ticker:      ticker,
				CompanyName: company,
				Catalyst:    cat,
				CardID:      cardID,
				VideoID:     videoID,
			})
		}
	}
	return nil, out, nil
}
