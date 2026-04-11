package main

import (
	"fmt"
	"os"
	"time"

	"github.com/annasblackhat/trading-alert/internal/api"
	"github.com/annasblackhat/trading-alert/internal/bot"
	"github.com/annasblackhat/trading-alert/internal/indicator"
	"github.com/annasblackhat/trading-alert/internal/notifier"
)

const (
	symbol        = "BTCUSDT"
	htfInterval   = "1h" // Higher Timeframe for Magic Lines
	ltfInterval   = "5m" // Lower Timeframe (where trigger lives)
	pollInterval  = 15 * time.Second
	pivotLength   = 15    // same as Pine pivotLength
	suckerCandles = 3     // same as Pine
	zoneTolerance = 0.003 // 0.3% as in Pine
)

func main() {
	fmt.Println("🚀 Modular Strategy Bot started (BTCUSDT 5m) - Long Only")
	fmt.Println("   Polling every 15s | Clean Architecture implementation")

	// 1. Setup API Client
	client := api.NewBinanceClient(symbol)

	// 2. Setup Indicator
	pitchfork := indicator.NewPitchfork(pivotLength, suckerCandles, zoneTolerance)

	// 3. Setup Notifier
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	tgChatID := os.Getenv("TELEGRAM_CHAT_ID")
	telegramNotifier := notifier.NewTelegramNotifier(tgToken, tgChatID)

	// 4. Setup Bot Orchestrator
	tradingBot := bot.NewBot(client, pitchfork, telegramNotifier, htfInterval, ltfInterval)

	// 5. Start Polling Loop
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Run once immediately before ticker
	tradingBot.RunCycle()

	for range ticker.C {
		tradingBot.RunCycle()
	}
}
