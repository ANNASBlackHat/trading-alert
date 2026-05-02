package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/annasblackhat/trading-alert/internal/api"
	"github.com/annasblackhat/trading-alert/internal/indicator"
	"github.com/annasblackhat/trading-alert/internal/model"
	"github.com/annasblackhat/trading-alert/internal/notifier"
)

type Bot struct {
	Client      api.MarketClient
	Indicator   indicator.Indicator
	Notifier    notifier.Notifier
	HTFInterval string
	LTFInterval string
	Name        string

	active          *model.ActiveSignal
	activeDirection string // "LONG" or "SHORT"
	lastCandleTime  int64
}

func NewBot(name string, client api.MarketClient, ind indicator.Indicator, notif notifier.Notifier, htf string, ltf string) *Bot {
	return &Bot{
		Name:        name,
		Client:      client,
		Indicator:   ind,
		Notifier:    notif,
		HTFInterval: htf,
		LTFInterval: ltf,
	}
}

func (b *Bot) RunCycle() {
	// 1. Fetch HTF
	htfKlines, err := b.Client.FetchKlines(b.HTFInterval, 500)
	if err != nil {
		slog.Error("HTF fetch error", "bot", b.Name, "err", err)
		return
	}

	// 2. Fetch LTF
	ltfKlines, err := b.Client.FetchKlines(b.LTFInterval, 100)
	if err != nil {
		slog.Error("LTF fetch error", "bot", b.Name, "err", err)
		return
	}

	// 3. Current price for proximity alerts
	currentPrice := b.Client.GetCurrentPrice()
	slog.Info("current price check", "bot", b.Name, slog.Float64("current_price", currentPrice))

	if len(ltfKlines) == 0 {
		slog.Error("LTF klines is empty", "bot", b.Name)
		return
	}

	// 4. Check for new completed candle
	latest := ltfKlines[len(ltfKlines)-1]
	if latest.OpenTime == b.lastCandleTime {
		// Same candle → only check proximity
		b.checkProximity(currentPrice)
		return
	}

	// New candle closed → process trigger
	b.lastCandleTime = latest.OpenTime

	signal := b.Indicator.Analyze(htfKlines, ltfKlines)
	slog.Info("indicator analysis finished", "bot", b.Name, slog.Bool("trigger", signal.Trigger))

	if signal.Trigger {
		slog.Info("SIGNAL DETECTED!", "bot", b.Name, "direction", signal.Direction)

		// Use strategy-specific message if provided, otherwise default format
		var msg string
		if signal.Message != "" {
			msg = fmt.Sprintf("[%s] %s", b.Name, signal.Message)
		} else {
			msg = fmt.Sprintf(`🚨 ALERT DETECTED! [%s]
Sucker move exhausted in Liquidity Zone
Green LTF candle closed

Entry: Buy Stop above %.2f
SL: below %.2f
TP Zone: %.2f (Red Magic Line)

Signal time: %s

Config:
High Interval: %v,
Low Interval: %v`, b.Name, signal.EntryPrice, signal.SLPrice, signal.TPPrice, time.Now().Format(time.RFC3339), b.HTFInterval, b.LTFInterval)
		}

		_ = b.Notifier.Send(msg)

		// Activate proximity monitoring
		b.active = &model.ActiveSignal{
			EntryPrice: signal.EntryPrice,
			TPPrice:    signal.TPPrice,
			SLPrice:    signal.SLPrice,
			SignalTime: time.Now(),
		}
		b.activeDirection = signal.Direction
	}

	// 5. Always check proximity if we have an active signal
	b.checkProximity(currentPrice)
}

func (b *Bot) checkProximity(current float64) {
	if b.active == nil {
		return
	}

	// Calculate progress — handle both LONG and SHORT directions
	var progress float64
	if b.activeDirection == "SHORT" {
		// SHORT: price should be going DOWN from entry toward TP
		if b.active.EntryPrice <= b.active.TPPrice {
			return // invalid state for short
		}
		progress = (b.active.EntryPrice - current) / (b.active.EntryPrice - b.active.TPPrice)
	} else {
		// LONG (default): price should be going UP from entry toward TP
		if b.active.TPPrice <= b.active.EntryPrice {
			return // invalid state for long
		}
		progress = (current - b.active.EntryPrice) / (b.active.TPPrice - b.active.EntryPrice)
	}

	dirEmoji := "📈"
	if b.activeDirection == "SHORT" {
		dirEmoji = "📉"
	}

	if progress >= 0.5 && !b.active.Notified50 {
		msg := fmt.Sprintf("%s 50%% to Target Area!\nPrice: %.2f | Progress: %.1f%%", dirEmoji, current, progress*100)
		_ = b.Notifier.Send(msg)
		b.active.Notified50 = true
	}
	if progress >= 0.7 && !b.active.Notified70 {
		msg := fmt.Sprintf("%s 70%% to Target Area (30%% closer)!\nPrice: %.2f | Progress: %.1f%%", dirEmoji, current, progress*100)
		_ = b.Notifier.Send(msg)
		b.active.Notified70 = true
	}
	if progress >= 0.9 && !b.active.Notified90 {
		msg := fmt.Sprintf("%s 90%% to Target Area (10%% closer)!\nPrice: %.2f | Progress: %.1f%%", dirEmoji, current, progress*100)
		_ = b.Notifier.Send(msg)
		b.active.Notified90 = true
	}

	// Reset when TP is reached
	if progress >= 1.0 {
		b.active = nil
		b.activeDirection = ""
	}
}
