package indicator

import (
	"log/slog"

	"github.com/annasblackhat/trading-alert/internal/model"
)

type Pitchfork struct {
	PivotLength   int
	SuckerCandles int
	ZoneTolerance float64
}

func NewPitchfork(pivotLength, suckerCandles int, zoneTolerance float64) *Pitchfork {
	return &Pitchfork{
		PivotLength:   pivotLength,
		SuckerCandles: suckerCandles,
		ZoneTolerance: zoneTolerance,
	}
}

func (p *Pitchfork) Analyze(htfKlines []model.Kline, ltfKlines []model.Kline) model.TradeSignal {
	var magicUpper float64
	var magicLower float64

	if ph, ok := latestPivotHigh(htfKlines, p.PivotLength, p.PivotLength); ok {
		magicUpper = ph
	}
	if pl, ok := latestPivotLow(htfKlines, p.PivotLength, p.PivotLength); ok {
		magicLower = pl
	}

	n := len(ltfKlines)
	if n < p.SuckerCandles+2 {
		return model.TradeSignal{Trigger: false}
	}

	current := ltfKlines[n-1] // just-closed 5m candle

	// isGreen = current candle must be green
	isGreen := current.Close > current.Open

	// Sucker Move: previous 'suckerCandles' must ALL be red
	isSuckerMove := true
	for i := n - 1 - p.SuckerCandles; i < n-1; i++ {
		if ltfKlines[i].Close >= ltfKlines[i].Open {
			isSuckerMove = false
			break
		}
	}

	// lowest low during the sucker move (previous N candles)
	lowestDuringDrop := ltfKlines[n-1-1].Low
	for i := n - 1 - p.SuckerCandles; i < n-1; i++ {
		if ltfKlines[i].Low < lowestDuringDrop {
			lowestDuringDrop = ltfKlines[i].Low
		}
	}

	// Zone touch check
	supZoneUpperBound := magicLower * (1 + p.ZoneTolerance)
	touchedZone := lowestDuringDrop <= supZoneUpperBound && lowestDuringDrop > 0

	trigger := isGreen && isSuckerMove && touchedZone
	slog.Info("pitchfork indicator analysis",
		slog.Bool("trigger", trigger),
		slog.Bool("is_green", isGreen),
		slog.Bool("is_sucker_move", isSuckerMove),
		slog.Bool("touched_zone", touchedZone),
		slog.Float64("lowest_during_drop", lowestDuringDrop),
		slog.Float64("magic_lower", magicLower),
		slog.Float64("sup_zone_upper_bound", supZoneUpperBound),
	)

	entryPrice := current.High
	slPrice := current.Low
	if lowestDuringDrop < slPrice {
		slPrice = lowestDuringDrop
	}

	return model.TradeSignal{
		Trigger:          trigger,
		EntryPrice:       entryPrice,
		SLPrice:          slPrice,
		TPPrice:          magicUpper,
		LowestDuringDrop: lowestDuringDrop,
	}
}

// ====================== PIVOT FUNCTIONS (exact match to ta.pivothigh/ta.pivotlow) ======================

func latestPivotHigh(klines []model.Kline, left, right int) (float64, bool) {
	n := len(klines)
	if n < left+right+1 {
		return 0, false
	}
	for i := n - right - 1; i >= left; i-- {
		val := klines[i].High
		isPivot := true
		// left bars
		for j := 1; j <= left; j++ {
			if klines[i-j].High >= val {
				isPivot = false
				break
			}
		}
		if !isPivot {
			continue
		}
		// right bars
		for j := 1; j <= right; j++ {
			if klines[i+j].High >= val {
				isPivot = false
				break
			}
		}
		if isPivot {
			return val, true
		}
	}
	return 0, false
}

func latestPivotLow(klines []model.Kline, left, right int) (float64, bool) {
	n := len(klines)
	if n < left+right+1 {
		return 0, false
	}
	for i := n - right - 1; i >= left; i-- {
		val := klines[i].Low
		isPivot := true
		for j := 1; j <= left; j++ {
			if klines[i-j].Low <= val {
				isPivot = false
				break
			}
		}
		if !isPivot {
			continue
		}
		for j := 1; j <= right; j++ {
			if klines[i+j].Low <= val {
				isPivot = false
				break
			}
		}
		if isPivot {
			return val, true
		}
	}
	return 0, false
}
