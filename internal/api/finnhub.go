package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/annasblackhat/trading-alert/internal/model"
)

// FinnhubClient implements MarketClient for stock data via Finnhub.io.
type FinnhubClient struct {
	symbol string
	apiKey string
}

// NewFinnhubClient creates a new FinnhubClient for the given stock symbol.
func NewFinnhubClient(symbol, apiKey string) *FinnhubClient {
	return &FinnhubClient{
		symbol: strings.ToUpper(symbol),
		apiKey: apiKey,
	}
}

// Symbol returns the trading symbol.
func (c *FinnhubClient) Symbol() string { return c.symbol }

// GetCurrentPrice fetches the latest quote price from Finnhub.
func (c *FinnhubClient) GetCurrentPrice() float64 {
	url := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", c.symbol, c.apiKey)
	body, err := c.doRequest(url)
	if err != nil {
		return 0
	}

	var resp struct {
		Current float64 `json:"c"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0
	}
	return resp.Current
}

// FetchKlines fetches historical candle bars from Finnhub.
func (c *FinnhubClient) FetchKlines(interval string, limit int) ([]model.Kline, error) {
	resolution, duration := mapIntervalToFinnhub(interval)

	// Pad the time window to account for weekends and non-trading hours
	// (Stock markets are only open 6.5 hours a day on weekdays).
	multiplier := 5.0
	if resolution == "D" || resolution == "W" {
		multiplier = 2.0
	}

	to := time.Now().Unix()
	from := to - int64(float64(limit)*duration.Seconds()*multiplier)

	url := fmt.Sprintf("https://finnhub.io/api/v1/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d&token=%s",
		c.symbol, resolution, from, to, c.apiKey)

	body, err := c.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("finnhub FetchKlines: %w | url: %s", err, url)
	}

	var resp struct {
		Open      []float64 `json:"o"`
		High      []float64 `json:"h"`
		Low       []float64 `json:"l"`
		Close     []float64 `json:"c"`
		Timestamp []int64   `json:"t"`
		Status    string    `json:"s"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("finnhub FetchKlines decode: %w", err)
	}

	if resp.Status != "ok" || len(resp.Timestamp) == 0 {
		return nil, fmt.Errorf("finnhub FetchKlines: no bars or invalid status=%s for symbol %s | url: %s", resp.Status, c.symbol, url)
	}

	klines := make([]model.Kline, len(resp.Timestamp))
	for i := 0; i < len(resp.Timestamp); i++ {
		klines[i] = model.Kline{
			OpenTime: resp.Timestamp[i] * 1000, // project expects milliseconds
			Open:     resp.Open[i],
			High:     resp.High[i],
			Low:      resp.Low[i],
			Close:    resp.Close[i],
		}
	}

	// Truncate to return only the requested limit (the most recent ones)
	if len(klines) > limit {
		klines = klines[len(klines)-limit:]
	}

	return klines, nil
}

// doRequest executes a GET request to Finnhub.
func (c *FinnhubClient) doRequest(url string) ([]byte, error) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// mapIntervalToFinnhub maps project intervals to Finnhub resolutions and approximate durations.
func mapIntervalToFinnhub(interval string) (string, time.Duration) {
	switch strings.ToLower(interval) {
	case "1m":
		return "1", time.Minute
	case "5m":
		return "5", 5 * time.Minute
	case "15m":
		return "15", 15 * time.Minute
	case "30m":
		return "30", 30 * time.Minute
	case "1h":
		return "60", time.Hour
	case "1d":
		return "D", 24 * time.Hour
	case "1w":
		return "W", 7 * 24 * time.Hour
	default:
		return "D", 24 * time.Hour
	}
}
