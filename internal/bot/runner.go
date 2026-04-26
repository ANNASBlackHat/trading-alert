package bot

import (
	"log/slog"
	"sync"
	"time"
)

// BotRunner manages multiple Bot instances, each on its own goroutine.
type BotRunner struct {
	bots         []*Bot
	pollInterval time.Duration
}

// NewBotRunner creates a BotRunner that will tick all bots at the given interval.
func NewBotRunner(pollInterval time.Duration, bots ...*Bot) *BotRunner {
	return &BotRunner{
		bots:         bots,
		pollInterval: pollInterval,
	}
}

// Run starts all bots concurrently. Blocks forever.
func (r *BotRunner) Run() {
	var wg sync.WaitGroup
	for _, b := range r.bots {
		wg.Add(1)
		go func(bot *Bot) {
			defer wg.Done()
			slog.Info("bot started", "name", bot.Name, "htf", bot.HTFInterval, "ltf", bot.LTFInterval)

			// Run once immediately before ticker
			bot.RunCycle()

			ticker := time.NewTicker(r.pollInterval)
			defer ticker.Stop()
			for range ticker.C {
				bot.RunCycle()
			}
		}(b)
	}
	wg.Wait()
}
