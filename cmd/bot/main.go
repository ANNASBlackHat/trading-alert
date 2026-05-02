package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/annasblackhat/trading-alert/internal/api"
	"github.com/annasblackhat/trading-alert/internal/bot"
	"github.com/annasblackhat/trading-alert/internal/config"
	"github.com/annasblackhat/trading-alert/internal/indicator"
	"github.com/annasblackhat/trading-alert/internal/notifier"
	"github.com/annasblackhat/trading-alert/internal/server"
	"github.com/annasblackhat/trading-alert/internal/telemetry"
)

func main() {
	fmt.Println("--------------------- TradingAlert Starting ---------------------")
	fmt.Println("                       Welcome Back, Master                      ")
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

	// Load YAML config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Multi-Bot Trading Alert started",
		"bot_count", len(cfg.Bots),
		"poll_interval_sec", cfg.PollIntervalSec,
	)

	// Shared Notifier (single Telegram channel for all bots)
	tgToken := cfg.Telegram.Token
	tgChatID := cfg.Telegram.ChatID
	// Allow env vars to override config file values
	if envToken := os.Getenv("TELEGRAM_BOT_TOKEN"); envToken != "" {
		tgToken = envToken
	}
	if envChatID := os.Getenv("TELEGRAM_CHAT_ID"); envChatID != "" {
		tgChatID = envChatID
	}
	telegramNotifier := notifier.NewTelegramNotifier(tgToken, tgChatID)

	// Resolve Alpaca credentials (from config, with env override)
	alpacaKey := cfg.Alpaca.APIKey
	alpacaSecret := cfg.Alpaca.SecretKey
	if envKey := os.Getenv("ALPACA_API_KEY"); envKey != "" {
		alpacaKey = envKey
	}
	if envSecret := os.Getenv("ALPACA_SECRET_KEY"); envSecret != "" {
		alpacaSecret = envSecret
	}

	// Build bots from config
	bots := make([]*bot.Bot, 0, len(cfg.Bots))
	for _, bc := range cfg.Bots {
		// Factory: resolve exchange client
		var client api.MarketClient
		switch bc.Exchange {
		case "binance":
			client = api.NewBinanceClient(bc.Symbol)
		case "alpaca":
			client = api.NewAlpacaClient(bc.Symbol, alpacaKey, alpacaSecret)
		default:
			slog.Error("unknown exchange, skipping bot", "exchange", bc.Exchange, "bot", bc.Name)
			continue
		}

		// Factory: resolve indicator
		var ind indicator.Indicator
		switch bc.Indicator {
		case "pitchfork":
			ind = indicator.NewPitchfork(bc.PivotLength, bc.SuckerCandles, bc.ZoneTolerance)
		case "session_range":
			ind = indicator.NewSessionRange(indicator.SessionRangeConfig{
				AsiaStart:      bc.AsiaStart,
				AsiaEnd:        bc.AsiaEnd,
				LondonStart:    bc.LondonStart,
				LondonEnd:      bc.LondonEnd,
				NYStart:        bc.NYStart,
				NYEnd:          bc.NYEnd,
				TightThreshold: bc.TightPips,
				EMALength:      bc.EMALength,
				RSILength:      bc.RSILength,
			})
		default:
			slog.Error("unknown indicator, skipping bot", "indicator", bc.Indicator, "bot", bc.Name)
			continue
		}

		b := bot.NewBot(bc.Name, client, ind, telegramNotifier, bc.HTFInterval, bc.LTFInterval)
		bots = append(bots, b)

		slog.Info("bot configured",
			"name", bc.Name,
			"symbol", bc.Symbol,
			"exchange", bc.Exchange,
			"htf", bc.HTFInterval,
			"ltf", bc.LTFInterval,
			"indicator", bc.Indicator,
		)
	}

	if len(bots) == 0 {
		slog.Error("no bots configured, exiting")
		os.Exit(1)
	}

	// Setup Telemetry & Server
	logProvider := telemetry.NewFileLogProvider(logFilePath)
	apiServer := server.NewServer("8080", logProvider)

	// Start API server in background
	go func() {
		if err := apiServer.Start(); err != nil {
			slog.Error("API Server crashed", slog.Any("error", err))
		}
	}()

	// Start all bots concurrently
	pollDuration := time.Duration(cfg.PollIntervalSec) * time.Second
	runner := bot.NewBotRunner(pollDuration, bots...)
	runner.Run() // blocks forever
}
