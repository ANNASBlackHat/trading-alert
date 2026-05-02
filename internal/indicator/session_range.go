package indicator

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/annasblackhat/trading-alert/internal/model"
)

// SessionRangeConfig holds all configurable parameters for the Session Range strategy.
type SessionRangeConfig struct {
	AsiaStart      int     // UTC hour (default 0)
	AsiaEnd        int     // UTC hour (default 6)
	LondonStart    int     // UTC hour (default 8)
	LondonEnd      int     // UTC hour (default 12)
	NYStart        int     // UTC hour (default 13)
	NYEnd          int     // UTC hour (default 17)
	TightThreshold float64 // max range (in price points) to qualify as "tight" (default 2.0)
	EMALength      int     // trend EMA period (default 50)
	RSILength      int     // momentum RSI period (default 14)
}

// SessionRange implements the Indicator interface for the
// Asian Range + London Sweep + NY Reversal strategy (ICT / Smart Money).
type SessionRange struct {
	cfg SessionRangeConfig

	// Internal state — persisted across RunCycle calls, reset daily.
	asiaHigh  float64
	asiaLow   float64
	asiaTight bool
	sweepType string // "bear_sweep" (London took high → LONG), "bull_sweep" (London took low → SHORT), ""
	lastDate  string // YYYY-MM-DD to detect day change and reset state
}

// NewSessionRange creates a new SessionRange indicator with defaults applied.
func NewSessionRange(cfg SessionRangeConfig) *SessionRange {
	// Apply defaults for zero values
	if cfg.AsiaEnd == 0 && cfg.AsiaStart == 0 {
		cfg.AsiaEnd = 6
	}
	if cfg.LondonStart == 0 {
		cfg.LondonStart = 8
	}
	if cfg.LondonEnd == 0 {
		cfg.LondonEnd = 12
	}
	if cfg.NYStart == 0 {
		cfg.NYStart = 13
	}
	if cfg.NYEnd == 0 {
		cfg.NYEnd = 17
	}
	if cfg.TightThreshold == 0 {
		cfg.TightThreshold = 2.0
	}
	if cfg.EMALength == 0 {
		cfg.EMALength = 50
	}
	if cfg.RSILength == 0 {
		cfg.RSILength = 14
	}

	return &SessionRange{cfg: cfg}
}

// Analyze processes HTF (context) and LTF (entry) klines to detect a Session Range signal.
//
// The strategy has three phases per day:
//  1. Asia (00:00–06:00 UTC) — identify range high/low, check tightness
//  2. London (08:00–12:00 UTC) — detect if London swept Asia high or low
//  3. NY (13:00–17:00 UTC) — generate entry signal if conditions met
func (s *SessionRange) Analyze(htfKlines []model.Kline, ltfKlines []model.Kline) model.TradeSignal {
	if len(ltfKlines) == 0 {
		return model.TradeSignal{Trigger: false}
	}

	// Determine current time from the latest LTF kline
	latestTime := time.UnixMilli(ltfKlines[len(ltfKlines)-1].OpenTime).UTC()
	currentDate := latestTime.Format("2006-01-02")
	currentHour := latestTime.Hour()

	// Reset state on new day
	if currentDate != s.lastDate {
		s.resetDaily(currentDate)
	}

	slog.Info("session_range phase check",
		"current_hour_utc", currentHour,
		"current_date", currentDate,
		"asia_high", s.asiaHigh,
		"asia_low", s.asiaLow,
		"asia_tight", s.asiaTight,
		"sweep_type", s.sweepType,
	)

	// Phase 1: Compute Asia range from LTF klines
	s.computeAsiaRange(ltfKlines)

	// Phase 2: Detect London sweep from LTF klines
	s.detectLondonSweep(ltfKlines)

	// Phase 3: Check for NY entry signal
	if !s.isInSession(currentHour, s.cfg.NYStart, s.cfg.NYEnd) {
		slog.Info("session_range: not in NY session, no signal",
			"current_hour", currentHour,
			"ny_start", s.cfg.NYStart,
			"ny_end", s.cfg.NYEnd,
		)
		return model.TradeSignal{Trigger: false}
	}

	// Gate: Asia must be tight
	if !s.asiaTight {
		slog.Info("session_range: Asia range not tight, skipping")
		return model.TradeSignal{Trigger: false}
	}

	// Gate: London must have swept
	if s.sweepType == "" {
		slog.Info("session_range: no London sweep detected, skipping")
		return model.TradeSignal{Trigger: false}
	}

	// Compute trend (EMA) and momentum (RSI) from HTF klines
	emaValue := computeEMA(htfKlines, s.cfg.EMALength)
	rsiValue := computeRSI(htfKlines, s.cfg.RSILength)
	currentPrice := ltfKlines[len(ltfKlines)-1].Close

	trendBias := "Neutral"
	if currentPrice > emaValue {
		trendBias = "Bullish"
	} else if currentPrice < emaValue {
		trendBias = "Bearish"
	}

	momentumBias := "Neutral"
	if rsiValue > 60 {
		momentumBias = "Strong"
	} else if rsiValue < 40 {
		momentumBias = "Weak"
	}

	slog.Info("session_range NY evaluation",
		"sweep_type", s.sweepType,
		"ema", emaValue,
		"rsi", rsiValue,
		"trend", trendBias,
		"momentum", momentumBias,
		"current_price", currentPrice,
		"asia_high", s.asiaHigh,
		"asia_low", s.asiaLow,
	)

	// Determine direction and validate alignment
	var direction string
	var entryPrice, slPrice, tpPrice float64

	switch s.sweepType {
	case "bear_sweep":
		// London took Asia High → expect NY reversal UP → LONG
		direction = "LONG"
		// Entry: current close (simplified from 15M confirmation candle logic)
		entryPrice = currentPrice
		// SL: below Asia Low
		slPrice = s.asiaLow
		// TP1: Asia High (the swept level)
		tpPrice = s.asiaHigh

	case "bull_sweep":
		// London took Asia Low → expect NY reversal DOWN → SHORT
		direction = "SHORT"
		entryPrice = currentPrice
		// SL: above Asia High
		slPrice = s.asiaHigh
		// TP1: Asia Low (the swept level)
		tpPrice = s.asiaLow
	}

	// Validate minimum R:R of 2:1
	risk := math.Abs(entryPrice - slPrice)
	reward := math.Abs(tpPrice - entryPrice)
	if risk == 0 || reward/risk < 2.0 {
		slog.Info("session_range: R:R below 2:1, skipping",
			"risk", risk,
			"reward", reward,
		)
		return model.TradeSignal{Trigger: false}
	}

	// Build strategy-specific alert message
	msg := fmt.Sprintf(`🔔 SESSION RANGE ALERT — %s

📊 Sweep: %s
📈 Trend (EMA %d): %s
📉 Momentum (RSI %d): %.1f (%s)

Asia Range: %.2f – %.2f (%s)
London Sweep: %s

Entry: %.2f
SL: %.2f
TP1: %.2f
R:R = %.1f:1

⏰ Signal at: %s`,
		direction,
		s.sweepType,
		s.cfg.EMALength, trendBias,
		s.cfg.RSILength, rsiValue, momentumBias,
		s.asiaLow, s.asiaHigh, func() string {
			if s.asiaTight {
				return "✅ TIGHT"
			}
			return "❌ WIDE"
		}(),
		s.sweepType,
		entryPrice,
		slPrice,
		tpPrice,
		reward/risk,
		latestTime.Format(time.RFC3339),
	)

	return model.TradeSignal{
		Trigger:    true,
		Direction:  direction,
		EntryPrice: entryPrice,
		SLPrice:    slPrice,
		TPPrice:    tpPrice,
		Message:    msg,
	}
}

// resetDaily clears all intra-day state for a new trading day.
func (s *SessionRange) resetDaily(date string) {
	slog.Info("session_range: resetting for new day", "date", date)
	s.asiaHigh = 0
	s.asiaLow = 0
	s.asiaTight = false
	s.sweepType = ""
	s.lastDate = date
}

// computeAsiaRange scans klines for bars within the Asia session window
// and computes the high/low range + tightness.
func (s *SessionRange) computeAsiaRange(klines []model.Kline) {
	if s.asiaHigh > 0 && s.asiaLow > 0 {
		// Already computed for today
		return
	}

	var high, low float64
	found := false

	for _, k := range klines {
		t := time.UnixMilli(k.OpenTime).UTC()
		h := t.Hour()
		if !s.isInSession(h, s.cfg.AsiaStart, s.cfg.AsiaEnd) {
			continue
		}
		if !found {
			high = k.High
			low = k.Low
			found = true
		} else {
			if k.High > high {
				high = k.High
			}
			if k.Low < low {
				low = k.Low
			}
		}
	}

	if !found {
		return
	}

	s.asiaHigh = high
	s.asiaLow = low

	rangeSize := high - low
	s.asiaTight = rangeSize <= s.cfg.TightThreshold

	slog.Info("session_range: Asia range computed",
		"high", high,
		"low", low,
		"range", rangeSize,
		"threshold", s.cfg.TightThreshold,
		"tight", s.asiaTight,
	)
}

// detectLondonSweep checks if London session broke above Asia High or below Asia Low.
func (s *SessionRange) detectLondonSweep(klines []model.Kline) {
	if s.sweepType != "" {
		// Already detected for today
		return
	}
	if s.asiaHigh == 0 || s.asiaLow == 0 {
		// Asia range not yet computed
		return
	}

	for _, k := range klines {
		t := time.UnixMilli(k.OpenTime).UTC()
		h := t.Hour()
		if !s.isInSession(h, s.cfg.LondonStart, s.cfg.LondonEnd) {
			continue
		}

		// Check for sweep of Asia High (bearish sweep → expect LONG at NY)
		if k.High > s.asiaHigh && s.sweepType == "" {
			s.sweepType = "bear_sweep"
			slog.Info("session_range: London swept Asia HIGH (bear_sweep → LONG expected)",
				"candle_high", k.High,
				"asia_high", s.asiaHigh,
			)
			return
		}

		// Check for sweep of Asia Low (bullish sweep → expect SHORT at NY)
		if k.Low < s.asiaLow && s.sweepType == "" {
			s.sweepType = "bull_sweep"
			slog.Info("session_range: London swept Asia LOW (bull_sweep → SHORT expected)",
				"candle_low", k.Low,
				"asia_low", s.asiaLow,
			)
			return
		}
	}
}

// isInSession checks if a UTC hour falls within [startH, endH).
// Handles midnight-crossing sessions (e.g., 22:00–06:00).
func (s *SessionRange) isInSession(h, startH, endH int) bool {
	if startH < endH {
		return h >= startH && h < endH
	}
	// Wraps midnight
	return h >= startH || h < endH
}

// ====================== TECHNICAL INDICATORS ======================

// computeEMA calculates an Exponential Moving Average of the closing prices.
func computeEMA(klines []model.Kline, length int) float64 {
	if len(klines) == 0 || length <= 0 {
		return 0
	}
	if len(klines) < length {
		// Not enough data — fall back to SMA of available bars
		sum := 0.0
		for _, k := range klines {
			sum += k.Close
		}
		return sum / float64(len(klines))
	}

	// Seed EMA with SMA of first 'length' bars
	sum := 0.0
	for i := 0; i < length; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(length)

	// EMA multiplier
	multiplier := 2.0 / float64(length+1)

	// Compute EMA for remaining bars
	for i := length; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}
	return ema
}

// computeRSI calculates the Relative Strength Index.
func computeRSI(klines []model.Kline, length int) float64 {
	if len(klines) < length+1 || length <= 0 {
		return 50 // neutral default when insufficient data
	}

	// Calculate initial average gains and losses
	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= length; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain += change
		} else {
			avgLoss += math.Abs(change)
		}
	}
	avgGain /= float64(length)
	avgLoss /= float64(length)

	// Smoothed RSI (Wilder's method) for remaining bars
	for i := length + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(length-1) + change) / float64(length)
			avgLoss = (avgLoss * float64(length-1)) / float64(length)
		} else {
			avgGain = (avgGain * float64(length-1)) / float64(length)
			avgLoss = (avgLoss*float64(length-1) + math.Abs(change)) / float64(length)
		}
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
