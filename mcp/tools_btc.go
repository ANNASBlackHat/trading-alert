package mcp

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/annasblackhat/trading-alert/internal/api"
)

// ── btc_get_agent_state ─────────────────────────────────────

// BtcAgentStateArgs has one optional field; include_prices attaches the live
// BTC price to the response.
type BtcAgentStateArgs struct {
	IncludePrices bool `json:"include_prices,omitempty" jsonschema:"attach the current BTC price (default false)"`
}

// BtcAgentState is the response for btc_get_agent_state.
type BtcAgentState struct {
	Found           bool             `json:"found"`
	Memory          *BtcAgentMemory  `json:"memory,omitempty"`
	OpenPredictions []map[string]any `json:"open_predictions,omitempty"`
	CurrentPrice    *float64         `json:"current_price,omitempty"`
	PriceAsOf       *time.Time       `json:"price_as_of,omitempty"`
	Unscored        int              `json:"unscored,omitempty"`
	Disclaimer      string           `json:"disclaimer"`
}

// BtcGetAgentState returns the btc_agent singleton memory plus, optionally,
// the live BTC price.
func BtcGetAgentState(ctx context.Context, _ *sdkmcp.CallToolRequest, in BtcAgentStateArgs) (*sdkmcp.CallToolResult, BtcAgentState, error) {
	s := instance()
	if s == nil {
		return nil, BtcAgentState{}, storeErr()
	}
	doc := s.btc.Collection("agent_memory")
	var mem BtcAgentMemory
	err := doc.FindOne(ctx, bson.M{}, options.FindOne()).Decode(&mem)

	out := BtcAgentState{Disclaimer: disclaimerText}

	if err == mongo.ErrNoDocuments {
		out.Found = false
		return nil, out, nil
	}
	if err != nil {
		return nil, out, fmt.Errorf("read agent_memory: %w", err)
	}

	out.Found = true
	out.Memory = &mem
	out.OpenPredictions = mem.OpenPredictions

	for _, p := range mem.OpenPredictions {
		if outcome, _ := p["outcome"].(string); outcome == "" {
			out.Unscored++
		}
	}

	if in.IncludePrices {
		btc := api.NewBinanceClient("BTCUSDT")
		price := btc.GetCurrentPrice()
		if price > 0 {
			out.CurrentPrice = &price
			now := time.Now().UTC()
			out.PriceAsOf = &now
		}
	}

	return nil, out, nil
}

// ── btc_list_predictions ─────────────────────────────────────

// BtcListPredictionsArgs filters the predictions collection.
type BtcListPredictionsArgs struct {
	Status     string `json:"status,omitempty" jsonschema:"filter: 'pending' (unscored), 'scored', or empty for all"`
	Days       int    `json:"days,omitempty" jsonschema:"look back N days (default 14)"`
	Confidence string `json:"confidence,omitempty" jsonschema:"filter by confidence: high, medium, low"`
	Limit      int    `json:"limit,omitempty" jsonschema:"max results (default 20, cap 100)"`
}

// BtcListPredictionsOut is the response for btc_list_predictions.
type BtcListPredictionsOut struct {
	Items      []PredictionRecord `json:"items"`
	Total      int                `json:"total"`
	Unscored   int                `json:"unscored"`
	Disclaimer string             `json:"disclaimer"`
}

// BtcListPredictions returns predictions filtered by status / date / confidence.
func BtcListPredictions(ctx context.Context, _ *sdkmcp.CallToolRequest, in BtcListPredictionsArgs) (*sdkmcp.CallToolResult, BtcListPredictionsOut, error) {
	s := instance()
	if s == nil {
		return nil, BtcListPredictionsOut{}, storeErr()
	}

	if in.Days <= 0 {
		in.Days = 14
	}
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.Limit > 100 {
		in.Limit = 100
	}

	filter := bson.M{}
	if in.Confidence != "" {
		filter["confidence"] = in.Confidence
	}

	coll := s.btc.Collection("predictions")

	// Scope to the lookback window via prediction_date (ISO strings sort
	// lexicographically, so $gte on the since string works).
	since := time.Now().UTC().AddDate(0, 0, -in.Days).Format("2006-01-02")
	filter["prediction_date"] = bson.M{"$gte": since}

	if in.Status == "pending" {
		filter["outcome"] = nil
	} else if in.Status == "scored" {
		filter["outcome"] = bson.M{"$ne": nil}
	}

	cursor, err := coll.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "prediction_date", Value: -1}}).
			SetLimit(int64(in.Limit)),
	)
	if err != nil {
		return nil, BtcListPredictionsOut{}, fmt.Errorf("query predictions: %w", err)
	}
	defer cursor.Close(ctx)

	var items []PredictionRecord
	if err := cursor.All(ctx, &items); err != nil {
		return nil, BtcListPredictionsOut{}, fmt.Errorf("decode predictions: %w", err)
	}

	unscored := 0
	for _, p := range items {
		if p.Outcome == nil {
			unscored++
		}
	}

	return nil, BtcListPredictionsOut{
		Items:      items,
		Total:      len(items),
		Unscored:   unscored,
		Disclaimer: disclaimerText,
	}, nil
}
