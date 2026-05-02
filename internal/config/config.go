package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// BotConfig defines the configuration for a single bot instance.
type BotConfig struct {
	Name          string  `yaml:"name"`
	Symbol        string  `yaml:"symbol"`
	Exchange      string  `yaml:"exchange"`       // "binance", "alpaca"
	HTFInterval   string  `yaml:"htf_interval"`
	LTFInterval   string  `yaml:"ltf_interval"`
	PivotLength   int     `yaml:"pivot_length"`
	SuckerCandles int     `yaml:"sucker_candles"`
	ZoneTolerance float64 `yaml:"zone_tolerance"`
	Indicator     string  `yaml:"indicator"`      // "pitchfork", "session_range"

	// Session Range specific config
	AsiaStart   int     `yaml:"asia_start,omitempty"`
	AsiaEnd     int     `yaml:"asia_end,omitempty"`
	LondonStart int     `yaml:"london_start,omitempty"`
	LondonEnd   int     `yaml:"london_end,omitempty"`
	NYStart     int     `yaml:"ny_start,omitempty"`
	NYEnd       int     `yaml:"ny_end,omitempty"`
	TightPips   float64 `yaml:"tight_pips,omitempty"`
	EMALength   int     `yaml:"ema_length,omitempty"`
	RSILength   int     `yaml:"rsi_length,omitempty"`
}

// TelegramConfig holds Telegram notifier credentials.
type TelegramConfig struct {
	Token  string `yaml:"token"`
	ChatID string `yaml:"chat_id"`
}

// AlpacaConfig holds Alpaca Markets API credentials.
type AlpacaConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
}

// AppConfig is the top-level application configuration.
type AppConfig struct {
	PollIntervalSec int            `yaml:"poll_interval_sec"`
	Telegram        TelegramConfig `yaml:"telegram"`
	Alpaca          AlpacaConfig   `yaml:"alpaca"`
	Bots            []BotConfig    `yaml:"bots"`
}

// Load reads and parses a YAML config file.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

