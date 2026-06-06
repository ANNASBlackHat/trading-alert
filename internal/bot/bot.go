package bot

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/annasblackhat/trading-alert/internal/api"
	"github.com/annasblackhat/trading-alert/internal/indicator"
	"github.com/annasblackhat/trading-alert/internal/model"
	"github.com/annasblackhat/trading-alert/internal/notifier"
	"github.com/annasblackhat/trading-alert/internal/target"
)

type Bot struct {
	Client      api.MarketClient
	Indicator   indicator.Indicator
	Notifier    notifier.Notifier
	TargetStore target.Store
	HTFInterval string
	LTFInterval string
	Name        string

	active          *model.ActiveSignal
	activeDirection string // "LONG" or "SHORT"
	lastCandleTime  int64
}

func NewBot(name string, client api.MarketClient, ind indicator.Indicator, notif notifier.Notifier, store target.Store, htf string, ltf string) *Bot {
	return &Bot{
		Name:        name,
		Client:      client,
		Indicator:   ind,
		Notifier:    notif,
		TargetStore: store,
		HTFInterval: htf,
		LTFInterval: ltf,
	}
}

func (b *Bot) RunCycle() {
	// 1. Current price check and target alerts are processed first
	currentPrice := b.Client.GetCurrentPrice()
	slog.Info("current price check", "bot", b.Name, slog.Float64("current_price", currentPrice))
	b.checkTargetAlerts(currentPrice)

	// 2. If no indicator is configured, we only monitor price targets and proximity alerts
	if b.Indicator == nil {
		b.checkProximity(currentPrice)
		return
	}

	// 3. Fetch HTF
	htfKlines, err := b.Client.FetchKlines(b.HTFInterval, 500)
	if err != nil {
		slog.Error("HTF fetch error", "bot", b.Name, "err", err)
		return
	}

	// 4. Fetch LTF
	ltfKlines, err := b.Client.FetchKlines(b.LTFInterval, 100)
	if err != nil {
		slog.Error("LTF fetch error", "bot", b.Name, "err", err)
		return
	}

	if len(ltfKlines) == 0 {
		slog.Error("LTF klines is empty", "bot", b.Name)
		return
	}

	// 5. Check for new completed candle
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

	// 6. Always check proximity if we have an active signal
	b.checkProximity(currentPrice)
}

func (b *Bot) checkTargetAlerts(current float64) {
	if b.TargetStore == nil || current == 0 {
		return
	}

	symbol := strings.ToUpper(b.Client.Symbol())
	targets, err := b.TargetStore.ListBySymbol(symbol)
	if err != nil {
		slog.Error("failed to list targets", "bot", b.Name, "symbol", symbol, "err", err)
		return
	}

	for _, t := range targets {
		if t.BotName != "" && t.BotName != b.Name {
			continue
		}

		if t.Type == target.TargetTypeTrailing {
			// Step 1: Check Activation
			if !t.IsActive {
				activated := false
				if t.Direction == target.DirectionDown { // price must reach >= ActivationPrice to activate
					if current >= t.ActivationPrice {
						activated = true
					}
				} else if t.Direction == target.DirectionUp { // price must fall <= ActivationPrice to activate
					if current <= t.ActivationPrice {
						activated = true
					}
				}

				if activated {
					t.IsActive = true
					t.ExtremePrice = current
					slog.Info("trailing stop target activated", "bot", b.Name, "target_id", t.ID, "activation_price", t.ActivationPrice, "extreme_price", current)
					if err := b.TargetStore.Update(t); err != nil {
						slog.Error("failed to update target activation state", "target_id", t.ID, "err", err)
					}
				} else {
					// Not active yet, skip tracking/trigger logic
					continue
				}
			}

			// Step 2: Initialize ExtremePrice on first check after activation if not set
			if t.ExtremePrice == 0 {
				t.ExtremePrice = current
				if err := b.TargetStore.Update(t); err != nil {
					slog.Error("failed to update target initial extreme price", "target_id", t.ID, "err", err)
				}
			}

			// Step 3: Update ExtremePrice
			extremeUpdated := false
			if t.Direction == target.DirectionDown {
				// Trailing down from peak (highest price since activation/setup)
				if current > t.ExtremePrice {
					t.ExtremePrice = current
					extremeUpdated = true
				}
			} else if t.Direction == target.DirectionUp {
				// Trailing up from trough (lowest price since activation/setup)
				if current < t.ExtremePrice {
					t.ExtremePrice = current
					extremeUpdated = true
				}
			}

			// Step 4: Evaluate Trigger
			triggered := false
			var threshold float64
			if t.Direction == target.DirectionDown {
				if t.TrailingPercent > 0 {
					threshold = t.ExtremePrice * (1.0 - t.TrailingPercent/100.0)
				} else {
					threshold = t.ExtremePrice - t.TrailingValue
				}
				if current <= threshold {
					triggered = true
				}
			} else if t.Direction == target.DirectionUp {
				if t.TrailingPercent > 0 {
					threshold = t.ExtremePrice * (1.0 + t.TrailingPercent/100.0)
				} else {
					threshold = t.ExtremePrice + t.TrailingValue
				}
				if current >= threshold {
					triggered = true
				}
			}

			if triggered {
				var msg string
				if t.Direction == target.DirectionDown {
					msg = fmt.Sprintf("🚨 TRAILING STOP ALERT [%s]\n%s dropped below trailing threshold!\nPeak High: %.2f\nThreshold: %.2f\nCurrent Price: %.2f",
						b.Name, symbol, t.ExtremePrice, threshold, current)
				} else {
					msg = fmt.Sprintf("🚨 TRAILING STOP ALERT [%s]\n%s rose above trailing threshold!\nTrough Low: %.2f\nThreshold: %.2f\nCurrent Price: %.2f",
						b.Name, symbol, t.ExtremePrice, threshold, current)
				}

				if t.Note != "" {
					msg = fmt.Sprintf("%s\n\nNote: %s", msg, t.Note)
				}

				if err := b.Notifier.Send(msg); err != nil {
					slog.Error("failed to send trailing stop alert", "bot", b.Name, "target_id", t.ID, "err", err)
				}
				if err := b.TargetStore.Delete(t.ID); err != nil {
					slog.Error("failed to delete triggered trailing target", "target_id", t.ID, "err", err)
				}
				continue
			}

			// If the extreme price updated but it didn't trigger, persist the new peak/trough
			if extremeUpdated {
				if err := b.TargetStore.Update(t); err != nil {
					slog.Error("failed to update target extreme price", "target_id", t.ID, "err", err)
				}
			}
			continue
		}

		currentState := target.NormalizeState(current, t.TargetPrice)
		if t.LastState == target.StateUnknown {
			t.LastState = currentState
			if err := b.TargetStore.Update(t); err != nil {
				slog.Error("failed to update initial target state", "target_id", t.ID, "err", err)
			}
			continue
		}

		triggered := false
		if t.Direction == target.DirectionUp && t.LastState == target.StateBelow && currentState == target.StateAbove {
			triggered = true
		}
		if t.Direction == target.DirectionDown && t.LastState == target.StateAbove && currentState == target.StateBelow {
			triggered = true
		}

		if triggered {
			msg := fmt.Sprintf("🎯 TARGET ALERT [%s]\n%s crossed %s %.2f\nCurrent: %.2f",
				b.Name, symbol, strings.ToUpper(string(t.Direction)), t.TargetPrice, current)
			if t.Note != "" {
				msg = fmt.Sprintf("%s\n\nNote: %s", msg, t.Note)
			}
			if err := b.Notifier.Send(msg); err != nil {
				slog.Error("failed to send target alert", "bot", b.Name, "target_id", t.ID, "err", err)
			}
			if err := b.TargetStore.Delete(t.ID); err != nil {
				slog.Error("failed to delete triggered target", "target_id", t.ID, "err", err)
			}
			continue
		}

		if t.LastState != currentState {
			t.LastState = currentState
			if err := b.TargetStore.Update(t); err != nil {
				slog.Error("failed to update target state", "target_id", t.ID, "err", err)
			}
		}
	}
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
