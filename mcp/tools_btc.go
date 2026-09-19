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

// ── btc_get_scoreboard ──────────────────────────────────────

// BtcScoreboardArgs controls the scoreboard aggregation.
type BtcScoreboardArgs struct {
	Days       int `json:"days,omitempty" jsonschema:"look back N days (default 30)"`
	MinSamples int `json:"min_samples,omitempty" jsonschema:"flag groups with fewer than this many scored calls (default 5)"`
}

// BtcGetScoreboard aggregates prediction accuracy by channel, confidence,
// and timeframe over the window. Only scored predictions are included.
func BtcGetScoreboard(ctx context.Context, _ *sdkmcp.CallToolRequest, in BtcScoreboardArgs) (*sdkmcp.CallToolResult, BtcScoreboardOut, error) {
	s := instance()
	if s == nil {
		return nil, BtcScoreboardOut{}, storeErr()
	}
	if in.Days <= 0 {
		in.Days = 30
	}
	if in.MinSamples <= 0 {
		in.MinSamples = 5
	}

	since := time.Now().UTC().AddDate(0, 0, -in.Days).Format("2006-01-02")

	// Pull all scored predictions in the window, aggregate client-side.
	// (The data set is small — a few hundred docs at most — so this is
	// cheap and avoids three separate Mongo aggregation pipelines.)
	filter := bson.M{
		"prediction_date": bson.M{"$gte": since},
		"outcome":         bson.M{"$ne": nil},
	}
	cursor, err := s.btc.Collection("predictions").Find(ctx, filter,
		options.Find().SetLimit(500),
	)
	if err != nil {
		return nil, BtcScoreboardOut{}, fmt.Errorf("query predictions for scoreboard: %w", err)
	}
	defer cursor.Close(ctx)

	var preds []PredictionRecord
	if err := cursor.All(ctx, &preds); err != nil {
		return nil, BtcScoreboardOut{}, fmt.Errorf("decode scored predictions: %w", err)
	}

	type agg struct {
		n       int
		correct int
		sumAcc  float64
	}
	byChannel := map[string]*agg{}
	byConfidence := map[string]*agg{}
	byTimeframe := map[string]*agg{}

	for _, p := range preds {
		isCorrect := p.Outcome != nil && (*p.Outcome == "correct" || *p.Outcome == "partial")
		bump := func(m map[string]*agg, key string) {
			if m[key] == nil {
				m[key] = &agg{}
			}
			m[key].n++
			if isCorrect {
				m[key].correct++
			}
			if p.AccuracyScore != nil {
				m[key].sumAcc += *p.AccuracyScore
			}
		}
		bump(byChannel, p.ChannelName)
		bump(byConfidence, p.Confidence)
		bump(byTimeframe, p.Timeframe)
	}

	out := BtcScoreboardOut{WindowDays: in.Days, Disclaimer: disclaimerText}
	toGroups := func(m map[string]*agg) []ScoreboardGroup {
		var gs []ScoreboardGroup
		for k, a := range m {
			hr := 0.0
			avg := 0.0
			if a.n > 0 {
				hr = round2(float64(a.correct) / float64(a.n))
			}
			if a.n > 0 {
				avg = round2(a.sumAcc / float64(a.n))
			}
			gs = append(gs, ScoreboardGroup{
				Key:         k,
				N:           a.n,
				Correct:     a.correct,
				HitRate:     &hr,
				AvgAccuracy: &avg,
			})
		}
		sort.Slice(gs, func(i, j int) bool {
			return gs[i].HitRate != nil && gs[j].HitRate != nil && *gs[i].HitRate > *gs[j].HitRate
		})
		return gs
	}
	out.ByChannel = toGroups(byChannel)
	out.ByConfidence = toGroups(byConfidence)
	out.ByTimeframe = toGroups(byTimeframe)

	for _, g := range out.ByChannel {
		if g.N < in.MinSamples {
			warn := fmt.Sprintf("channel %q has only %d scored calls; hit-rate is not statistically meaningful", g.Key, g.N)
			out.MinSampleWarning = &warn
			break
		}
	}
	if len(preds) < in.MinSamples {
		warn := fmt.Sprintf("only %d scored predictions in the window; treat all rates as anecdotal", len(preds))
		out.MinSampleWarning = &warn
	}

	return nil, out, nil
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

// ── btc_technique_stats ─────────────────────────────────────

// BtcTechniqueStatsArgs controls the technique-ledger query.
type BtcTechniqueStatsArgs struct {
	MinTimesUsed int `json:"min_times_used,omitempty" jsonschema:"only include techniques used at least this many times (default 3)"`
	Limit        int `json:"limit,omitempty" jsonschema:"max rows (default 50, cap 200)"`
}

// BtcGetTechniqueStats returns the technique ledger sorted by hit-rate desc,
// with a low-sample flag on techniques used fewer than 5 times.
func BtcGetTechniqueStats(ctx context.Context, _ *sdkmcp.CallToolRequest, in BtcTechniqueStatsArgs) (*sdkmcp.CallToolResult, BtcTechniqueStatsOut, error) {
	s := instance()
	if s == nil {
		return nil, BtcTechniqueStatsOut{}, storeErr()
	}
	if in.MinTimesUsed <= 0 {
		in.MinTimesUsed = 3
	}
	if in.Limit <= 0 {
		in.Limit = 50
	}
	if in.Limit > 200 {
		in.Limit = 200
	}

	coll := s.btc.Collection("technique_ledger")
	filter := bson.M{"times_used": bson.M{"$gte": in.MinTimesUsed}}
	cursor, err := coll.Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "hit_rate", Value: -1}}).
			SetLimit(int64(in.Limit)),
	)
	if err != nil {
		return nil, BtcTechniqueStatsOut{}, fmt.Errorf("query technique_ledger: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []TechniqueLedgerEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, BtcTechniqueStatsOut{}, fmt.Errorf("decode technique_ledger: %w", err)
	}

	// The LLM's free-text technique names vary in casing across runs
	// ("Bear flag pattern" vs "Bear Flag pattern"), which creates
	// duplicate ledger rows. Aggregate them case-insensitively.
	byName := map[string]*TechniqueStat{}
	for _, e := range entries {
		key := strings.ToLower(e.TechniqueName)
		if byName[key] == nil {
			byName[key] = &TechniqueStat{TechniqueName: e.TechniqueName}
		}
		t := byName[key]
		t.TimesUsed += e.TimesUsed
		t.CorrectCalls += e.CorrectCalls
		if e.HitRate > t.HitRate {
			t.HitRate = e.HitRate
		}
		if e.BestMarketCondition != nil && t.BestMarketCondition == nil {
			t.BestMarketCondition = e.BestMarketCondition
		}
	}

	out := BtcTechniqueStatsOut{Disclaimer: disclaimerText}
	for _, t := range byName {
		if t.TimesUsed > 0 {
			t.HitRate = round2(float64(t.CorrectCalls) / float64(t.TimesUsed))
		}
		t.LowSample = t.TimesUsed < 5
		out.Techniques = append(out.Techniques, *t)
	}
	sort.Slice(out.Techniques, func(i, j int) bool {
		return out.Techniques[i].HitRate > out.Techniques[j].HitRate
	})
	return nil, out, nil
}

// ── btc_search_analyses ─────────────────────────────────────

// BtcSearchAnalysesArgs filters the daily_analyses collection.
type BtcSearchAnalysesArgs struct {
	SinceDays         int    `json:"since_days,omitempty" jsonschema:"only analyses from N days back (default 7)"`
	ChannelID         string `json:"channel_id,omitempty" jsonschema:"filter by channel_id"`
	MarketStructure   string `json:"market_structure,omitempty" jsonschema:"filter: bullish, bearish, ranging, unclear"`
	Technique         string `json:"technique,omitempty" jsonschema:"only analyses that used a technique whose name contains this string"`
	IncludeTranscript bool   `json:"include_transcript,omitempty" jsonschema:"include the raw_transcription field (default false; it can be up to 8000 chars)"`
	Limit             int    `json:"limit,omitempty" jsonschema:"max results (default 20, cap 100)"`
}

// BtcSearchAnalyses returns daily analyses filtered by date, channel,
// market structure, and technique.
func BtcSearchAnalyses(ctx context.Context, _ *sdkmcp.CallToolRequest, in BtcSearchAnalysesArgs) (*sdkmcp.CallToolResult, BtcSearchAnalysesOut, error) {
	s := instance()
	if s == nil {
		return nil, BtcSearchAnalysesOut{}, storeErr()
	}
	if in.SinceDays <= 0 {
		in.SinceDays = 7
	}
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.Limit > 100 {
		in.Limit = 100
	}

	since := time.Now().UTC().AddDate(0, 0, -in.SinceDays).Format("2006-01-02")
	filter := bson.M{"analysis_date": bson.M{"$gte": since}}
	if in.ChannelID != "" {
		filter["channel_id"] = in.ChannelID
	}
	if in.MarketStructure != "" {
		filter["market_structure"] = in.MarketStructure
	}
	if in.Technique != "" {
		filter["techniques_used.name"] = bson.M{"$regex": in.Technique, "$options": "i"}
	}

	// Exclude raw_transcription by default; the struct's bson tag already
	// omits it, so a plain Find + Decode is enough.
	coll := s.btc.Collection("daily_analyses")
	findOpts := options.Find().
		SetSort(bson.D{{Key: "analysis_date", Value: -1}}).
		SetLimit(int64(in.Limit))

	// When the transcript is not wanted, project it out explicitly so even
	// a future struct change can't leak it.
	if !in.IncludeTranscript {
		findOpts = findOpts.SetProjection(bson.M{"raw_transcription": 0})
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, BtcSearchAnalysesOut{}, fmt.Errorf("query daily_analyses: %w", err)
	}
	defer cursor.Close(ctx)

	var items []DailyAnalysis
	if err := cursor.All(ctx, &items); err != nil {
		return nil, BtcSearchAnalysesOut{}, fmt.Errorf("decode daily_analyses: %w", err)
	}

	// When the transcript is not requested, strip it from the response.
	if !in.IncludeTranscript {
		for i := range items {
			items[i].RawTranscription = nil
		}
	}

	return nil, BtcSearchAnalysesOut{
		Items:      items,
		Total:      len(items),
		Disclaimer: disclaimerText,
	}, nil
}
