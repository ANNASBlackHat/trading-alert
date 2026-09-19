package mcp

import (
	"fmt"
	"strings"
)

// coerceStr turns a heterogeneous BSON-decoded value (string, number, array,
// or nil) into a plain string. An array is joined with ", " so that a single
// malformed document cannot break an entire tool response.
func coerceStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return fmt.Sprintf("%g", t)
	case int32:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			if e != nil {
				if s := coerceStr(e); s != "" {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// toStrList turns a heterogeneous BSON-decoded value into a string slice:
// an array becomes the slice, a scalar becomes a single-element slice, and
// nil becomes an empty slice.
func toStrList(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if e != nil {
				out = append(out, coerceStr(e))
			}
		}
		return out
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	default:
		return []string{coerceStr(t)}
	}
}

// hydrateStockCard builds a StockCard from a raw document map. The LLM
// generates these fields with inconsistent shapes across runs (string vs
// number vs array), so we coerce rather than strict-decode, and we read the
// stable fields from the map too — no bson struct decode at all.
func hydrateStockCard(m map[string]any) StockCard {
	var c StockCard

	c.CardID = coerceStr(m["card_id"])
	c.VideoID = coerceStr(m["video_id"])
	c.SchemaVersion = coerceStr(m["schema_version"])

	if tv, ok := m["ticker"]; ok && tv != nil {
		s := coerceStr(tv)
		c.Ticker = &s
	}
	c.TickerUnresolved = toBool(m["ticker_unresolved"])
	c.CompanyName = coerceStr(m["company_name"])
	c.Sector = coerceStr(m["sector"])
	c.SpeakerName = coerceStr(m["speaker_name"])

	if tv, ok := m["timestamp_in_video"]; ok && tv != nil {
		s := coerceStr(tv)
		c.TimestampInVideo = &s
	}

	c.Depth = coerceStr(m["depth"])
	c.Stance = coerceStr(m["stance"])
	c.ConfidenceLanguage = coerceStr(m["confidence_language"])
	c.ReasoningType = toStrList(m["reasoning_type"])
	c.ReasoningSummary = coerceStr(m["reasoning_summary"])
	c.Timeframe = coerceStr(m["timeframe"])
	c.DirectQuote = coerceStr(m["direct_quote"])

	c.PastCatalystsReferenced = toStrList(m["past_catalysts_referenced"])
	c.UpcomingCatalysts = toStrList(m["upcoming_catalysts"])

	if tv, ok := m["historical_price_narrative"]; ok && tv != nil {
		s := coerceStr(tv)
		c.HistoricalPriceNarrative = &s
	}
	if tv, ok := m["price_at_time_of_recording"]; ok && tv != nil {
		s := coerceStr(tv)
		c.PriceAtTimeOfRecording = &s
	}
	if tv, ok := m["forward_price_target_or_level"]; ok && tv != nil {
		s := coerceStr(tv)
		c.ForwardPriceTargetOrLevel = &s
	}

	return c
}

// toBool coerces a BSON bool-like value to bool.
func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	default:
		return false
	}
}
