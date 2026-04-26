package main

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/annasblackhat/trading-alert/internal/api"
	"github.com/annasblackhat/trading-alert/internal/bot"
	"github.com/annasblackhat/trading-alert/internal/indicator"
	"github.com/annasblackhat/trading-alert/internal/notifier"
	"github.com/annasblackhat/trading-alert/internal/server"
	"github.com/annasblackhat/trading-alert/internal/telemetry"
)

const (
	symbol         = "BTCUSDT"
	htfInterval    = "1h"  // Higher Timeframe for Magic Lines
	ltfInterval    = "5m"  // Lower Timeframe (where trigger lives)
	htfInterval2   = "1d"  // Higher Timeframe for Magic Lines
	ltfInterval2   = "15m" // Lower Timeframe (where trigger lives)
	pollInterval   = 15 * time.Second
	pivotLength    = 15    // same as Pine pivotLength
	suckerCandles  = 3     // same as Pine
	zoneTolerance  = 0.003 // 0.3% as in Pine
	zoneTolerance2 = 0.003 // 0.3% as in Pine
)

func main() {
	// Load .env file. If it doesn't exist, we just rely on system environment variables.
	if err := godotenv.Load(); err != nil {
		// Just silently proceed if no .env
	}

	// Setup lumberjack log rotation
	logFilePath := "bot.log"
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   // days
		Compress:   true, // disabled by default
	}
	defer lumberjackLogger.Close()

	// Setup structured logging with MultiWriter
	multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Modular Strategy Bot started (BTCUSDT 5m) - Long Only", "polling_interval", pollInterval)
	slog.Info("Clean Architecture implementation and multi-writer log rotation enabled")

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

	// 5. Setup Telemetry & Server
	logProvider := telemetry.NewFileLogProvider(logFilePath)
	apiServer := server.NewServer("8080", logProvider)

	// Start API server in background
	go func() {
		if err := apiServer.Start(); err != nil {
			slog.Error("API Server crashed", slog.Any("error", err))
		}
	}()

	// 6. Start Polling Loop
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Run once immediately before ticker
	tradingBot.RunCycle()

	for range ticker.C {
		tradingBot.RunCycle()
	}
}
