// Package mcp implements the Trading Agent Memory MCP server.
//
// It exposes both the btc_agent and stocks_agent MongoDB databases to any
// AI chat agent, plus the live price infrastructure from internal/api.
//
// NOTE: All tool outputs are third-party analyst opinions and historical
// scores, not financial advice.
package mcp

import "time"

// ── btc_agent database ───────────────────────────────────────

// Technique mirrors the embedded Technique object in daily_analyses.
type Technique struct {
	Name      string  `bson:"name" json:"name"`
	Timeframe *string `bson:"timeframe,omitempty" json:"timeframe,omitempty"`
	Signal    string  `bson:"signal" json:"signal"`
}

// ScenarioPrediction mirrors the embedded scenario in daily_analyses.predictions.
type ScenarioPrediction struct {
	Label       string   `bson:"label" json:"label"`
	Condition   string   `bson:"condition" json:"condition"`
	Target      *float64 `bson:"target,omitempty" json:"target,omitempty"`
	Probability *string  `bson:"probability,omitempty" json:"probability,omitempty"`
}

// PredictionsBlock mirrors the embedded predictions object in daily_analyses.
type PredictionsBlock struct {
	Primary   *map[string]any      `bson:"primary,omitempty" json:"primary,omitempty"`
	Scenarios []ScenarioPrediction `bson:"scenarios,omitempty" json:"scenarios,omitempty"`
	ShortTerm *string              `bson:"short_term,omitempty" json:"short_term,omitempty"`
	LongTerm  *string              `bson:"long_term,omitempty" json:"long_term,omitempty"`
	Narrative *string              `bson:"narrative,omitempty" json:"narrative,omitempty"`
}

// KeyLevels mirrors the embedded key_levels object.
type KeyLevels struct {
	Support    []float64 `bson:"support" json:"support"`
	Resistance []float64 `bson:"resistance" json:"resistance"`
}

// DailyAnalysis mirrors a doc in btc_agent.daily_analyses.
type DailyAnalysis struct {
	VideoID           string           `bson:"video_id" json:"video_id"`
	ChannelID         string           `bson:"channel_id" json:"channel_id"`
	ChannelName       string           `bson:"channel_name" json:"channel_name"`
	AnalysisDate      string           `bson:"analysis_date" json:"analysis_date"`
	SchemaVersion     string           `bson:"schema_version" json:"schema_version"`
	BtcPriceMentioned *float64         `bson:"btc_price_mentioned,omitempty" json:"btc_price_mentioned,omitempty"`
	MarketStructure   *string          `bson:"market_structure,omitempty" json:"market_structure,omitempty"`
	TechniquesUsed    []Technique      `bson:"techniques_used,omitempty" json:"techniques_used,omitempty"`
	Predictions       PredictionsBlock `bson:"predictions" json:"predictions"`
	KeyLevels         KeyLevels        `bson:"key_levels" json:"key_levels"`
	Catalysts         []string         `bson:"catalysts,omitempty" json:"catalysts,omitempty"`
	ContrarianView    *string          `bson:"contrarian_view,omitempty" json:"contrarian_view,omitempty"`
	RawTranscription  *string          `bson:"raw_transcription,omitempty" json:"-"` // excluded unless requested
}

// PredictionRecord mirrors a doc in btc_agent.predictions.
type PredictionRecord struct {
	VideoID        string   `bson:"video_id" json:"video_id"`
	ChannelName    string   `bson:"channel_name" json:"channel_name"`
	PredictionDate string   `bson:"prediction_date" json:"prediction_date"`
	Direction      string   `bson:"direction" json:"direction"`
	Target         float64  `bson:"target" json:"target"`
	Timeframe      string   `bson:"timeframe" json:"timeframe"`
	TargetDate     string   `bson:"target_date" json:"target_date"`
	Confidence     string   `bson:"confidence" json:"confidence"`
	Invalidation   *string  `bson:"invalidation,omitempty" json:"invalidation,omitempty"`
	ActualPrice    *float64 `bson:"actual_price,omitempty" json:"actual_price,omitempty"`
	Outcome        *string  `bson:"outcome,omitempty" json:"outcome,omitempty"`
	AccuracyScore  *float64 `bson:"accuracy_score,omitempty" json:"accuracy_score,omitempty"`
}

// TechniqueLedgerEntry mirrors a doc in btc_agent.technique_ledger.
type TechniqueLedgerEntry struct {
	TechniqueName       string           `bson:"technique_name" json:"technique_name"`
	Description         *string          `bson:"description,omitempty" json:"description,omitempty"`
	TimesUsed           int              `bson:"times_used" json:"times_used"`
	CorrectCalls        int              `bson:"correct_calls" json:"correct_calls"`
	HitRate             float64          `bson:"hit_rate" json:"hit_rate"`
	BestMarketCondition *string          `bson:"best_market_condition,omitempty" json:"best_market_condition,omitempty"`
	RecentExamples      []map[string]any `bson:"recent_examples,omitempty" json:"recent_examples,omitempty"`
	LastUpdated         string           `bson:"last_updated" json:"last_updated"`
}

// BtcAgentMemory mirrors the singleton doc in btc_agent.agent_memory.
type BtcAgentMemory struct {
	LastUpdated        string           `bson:"last_updated" json:"last_updated"`
	MarketNarrative    string           `bson:"market_narrative" json:"market_narrative"`
	KeyLevelsConsensus KeyLevels        `bson:"key_levels_consensus" json:"key_levels_consensus"`
	TechniqueInsights  []string         `bson:"technique_insights" json:"technique_insights"`
	ChannelReliability map[string]any   `bson:"channel_reliability" json:"channel_reliability"`
	OpenPredictions    []map[string]any `bson:"open_predictions" json:"open_predictions"`
	AgentCurrentView   string           `bson:"agent_current_view" json:"agent_current_view"`
	AgentReflection    string           `bson:"agent_reflection" json:"agent_reflection"`
}

// BtcAgentOpinion mirrors a doc in btc_agent.agent_opinions.
type BtcAgentOpinion struct {
	OpinionDate     string   `bson:"opinion_date" json:"opinion_date"`
	Direction       string   `bson:"direction" json:"direction"`
	PriceTarget     *float64 `bson:"price_target,omitempty" json:"price_target,omitempty"`
	Reasoning       string   `bson:"reasoning" json:"reasoning"`
	TechniquesCited []string `bson:"techniques_cited" json:"techniques_cited"`
	ActualPrice     *float64 `bson:"actual_price,omitempty" json:"actual_price,omitempty"`
	Outcome         *string  `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Reflection      *string  `bson:"reflection,omitempty" json:"reflection,omitempty"`
}

// ProcessedVideoBtc mirrors a doc in btc_agent.processed_videos.
type ProcessedVideoBtc struct {
	VideoID     string `bson:"video_id" json:"video_id"`
	ChannelID   string `bson:"channel_id" json:"channel_id"`
	ChannelName string `bson:"channel_name" json:"channel_name"`
	Title       string `bson:"title" json:"title"`
	PublishedAt string `bson:"published_at" json:"published_at"`
	ProcessedAt string `bson:"processed_at" json:"processed_at"`
	Status      string `bson:"status" json:"status"`
}

// ── stocks_agent database ─────────────────────────────────────

// Channel mirrors a doc in stocks_agent.channels.
type Channel struct {
	ChannelID string `bson:"channel_id" json:"channel_id"`
	Name      string `bson:"name" json:"name"`
}

// ProcessedVideoStocks mirrors a doc in stocks_agent.processed_videos.
type ProcessedVideoStocks struct {
	VideoID       string  `bson:"video_id" json:"video_id"`
	SchemaVersion string  `bson:"schema_version" json:"schema_version"`
	ChannelID     string  `bson:"channel_id" json:"channel_id"`
	PublishDate   string  `bson:"publish_date" json:"publish_date"`
	ProcessedAt   string  `bson:"processed_at" json:"processed_at"`
	Status        string  `bson:"status" json:"status"`
	Format        *string `bson:"format,omitempty" json:"format,omitempty"`
	ContentStyle  *string `bson:"content_style,omitempty" json:"content_style,omitempty"`
}

// Speaker mirrors a doc in stocks_agent.speakers.
type Speaker struct {
	VideoID     string  `bson:"video_id" json:"video_id"`
	Name        string  `bson:"name" json:"name"`
	Role        string  `bson:"role" json:"role"`
	Affiliation *string `bson:"affiliation,omitempty" json:"affiliation,omitempty"`
}

// StockCard mirrors a doc in stocks_agent.stock_cards.
//
// LLM-generated fields (depth, stance, confidence_language, timeframe,
// reasoning_type, catalyst lists) are read via hydrateStockCard from a raw
// document map — see flex.go — because their shape varies across runs. The
// bson tags below are informational; hydrateStockCard is the actual decode
// path used by the tools.
type StockCard struct {
	CardID                    string   `json:"card_id"`
	VideoID                   string   `json:"video_id"`
	SchemaVersion             string   `json:"schema_version"`
	Ticker                    *string  `json:"ticker,omitempty"`
	TickerUnresolved          bool     `json:"ticker_unresolved,omitempty"`
	CompanyName               string   `json:"company_name"`
	Sector                    string   `json:"sector"`
	SpeakerName               string   `json:"speaker_name"`
	TimestampInVideo          *string  `json:"timestamp_in_video,omitempty"`
	Depth                     string   `json:"depth"`
	Stance                    string   `json:"stance"`
	ConfidenceLanguage        string   `json:"confidence_language"`
	ReasoningType             []string `json:"reasoning_type"`
	ReasoningSummary          string   `json:"reasoning_summary"`
	HistoricalPriceNarrative  *string  `json:"historical_price_narrative,omitempty"`
	PriceAtTimeOfRecording    *string  `json:"price_at_time_of_recording,omitempty"`
	ForwardPriceTargetOrLevel *string  `json:"forward_price_target_or_level,omitempty"`
	PastCatalystsReferenced   []string `json:"past_catalysts_referenced,omitempty"`
	UpcomingCatalysts         []string `json:"upcoming_catalysts,omitempty"`
	Timeframe                 string   `json:"timeframe"`
	DirectQuote               string   `json:"direct_quote"`
}

// ConceptCard mirrors a doc in stocks_agent.concept_cards.
type ConceptCard struct {
	CardID             string  `bson:"card_id" json:"card_id"`
	VideoID            string  `bson:"video_id" json:"video_id"`
	SchemaVersion      string  `bson:"schema_version" json:"schema_version"`
	Concept            string  `bson:"concept" json:"concept"`
	ConceptCategory    string  `bson:"concept_category" json:"concept_category"`
	ExplanationSummary string  `bson:"explanation_summary" json:"explanation_summary"`
	AppliesToSector    *string `bson:"applies_to_sector,omitempty" json:"applies_to_sector,omitempty"`
	TimestampInVideo   *string `bson:"timestamp_in_video,omitempty" json:"timestamp_in_video,omitempty"`
}

// TickerLookupEntry mirrors a doc in stocks_agent.ticker_lookup.
type TickerLookupEntry struct {
	CompanyNameNormalized string `bson:"company_name_normalized" json:"company_name_normalized"`
	Ticker                string `bson:"ticker" json:"ticker"`
	Exchange              string `bson:"exchange" json:"exchange"`
}

// StocksAgentMemory mirrors the singleton doc in stocks_agent.agent_memory.
type StocksAgentMemory struct {
	LastUpdated        string           `bson:"last_updated" json:"last_updated"`
	MarketNarrative    string           `bson:"market_narrative" json:"market_narrative"`
	SectorsConsensus   map[string]any   `bson:"sectors_consensus" json:"sectors_consensus"`
	TechniqueInsights  []string         `bson:"technique_insights" json:"technique_insights"`
	ChannelReliability map[string]any   `bson:"channel_reliability" json:"channel_reliability"`
	OpenPredictions    []map[string]any `bson:"open_predictions" json:"open_predictions"`
	AgentCurrentView   string           `bson:"agent_current_view" json:"agent_current_view"`
	AgentReflection    string           `bson:"agent_reflection" json:"agent_reflection"`
}

// StocksAgentOpinion mirrors a doc in stocks_agent.agent_opinions.
type StocksAgentOpinion struct {
	OpinionDate     string   `bson:"opinion_date" json:"opinion_date"`
	Direction       string   `bson:"direction" json:"direction"`
	PriceTarget     *float64 `bson:"price_target,omitempty" json:"price_target,omitempty"`
	Reasoning       string   `bson:"reasoning" json:"reasoning"`
	TechniquesCited []string `bson:"techniques_cited" json:"techniques_cited"`
	ActualPrice     *float64 `bson:"actual_price,omitempty" json:"actual_price,omitempty"`
	Outcome         *string  `bson:"outcome,omitempty" json:"outcome,omitempty"`
}

// ── Tool result shapes (returned to the chat agent) ─────────

// WithLivePrice attaches the current market price to open predictions
// so the agent can answer "how is this call performing so far?".
type OpenPredictionWithPrice struct {
	Raw          map[string]any `json:"prediction"`
	CurrentPrice *float64       `json:"current_price,omitempty"`
	AsOf         time.Time      `json:"as_of,omitempty"`
}

// StanceBreakdown summarizes how many cards lean bull/bear/neutral.
type StanceBreakdown struct {
	Bull    int `json:"bull"`
	Bear    int `json:"bear"`
	Neutral int `json:"neutral"`
	Other   int `json:"other,omitempty"`
}

// DataQuality flags surface caveats so the LLM never overstates small samples.
type DataQuality struct {
	UnresolvedTickerCards int     `json:"unresolved_ticker_cards,omitempty"`
	UnscoredPredictions   int     `json:"unscored_predictions,omitempty"`
	MinSampleWarning      *string `json:"min_sample_warning,omitempty"`
}
