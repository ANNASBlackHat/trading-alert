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

const alpacaBaseURL = "https://data.alpaca.markets/v2"

// AlpacaClient implements MarketClient for US stocks via Alpaca Markets API.
type AlpacaClient struct {
	symbol    string
	apiKey    string
	secretKey string
}

// NewAlpacaClient creates a new AlpacaClient for the given stock symbol.
func NewAlpacaClient(symbol, apiKey, secretKey string) *AlpacaClient {
	return &AlpacaClient{
		symbol:    strings.ToUpper(symbol),
		apiKey:    apiKey,
		secretKey: secretKey,
	}
}

// Symbol returns the trading symbol.
func (c *AlpacaClient) Symbol() string { return c.symbol }

// FetchKlines fetches historical bars from Alpaca.
// interval uses the same short format as Binance (e.g. "1h", "15m", "1d")
// which is mapped to Alpaca's format (e.g. "1Hour", "15Min", "1Day").
func (c *AlpacaClient) FetchKlines(interval string, limit int) ([]model.Kline, error) {
	alpacaTF := mapIntervalToAlpaca(interval)

	url := fmt.Sprintf("%s/stocks/bars?symbols=%s&timeframe=%s&limit=%d&sort=asc",
		alpacaBaseURL, c.symbol, alpacaTF, limit)

	body, err := c.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("alpaca FetchKlines: %w | url: %s", err, url)
	}

	var resp struct {
		Bars map[string][]struct {
			Open  float64 `json:"o"`
			High  float64 `json:"h"`
			Low   float64 `json:"l"`
			Close float64 `json:"c"`
			Time  string  `json:"t"`
		} `json:"bars"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("alpaca FetchKlines decode: %w", err)
	}

	bars, ok := resp.Bars[c.symbol]
	if !ok {
		return nil, fmt.Errorf("alpaca FetchKlines: no bars for symbol %s", c.symbol)
	}

	klines := make([]model.Kline, len(bars))
	for i, b := range bars {
		t, _ := time.Parse(time.RFC3339, b.Time)
		klines[i] = model.Kline{
			OpenTime: t.UnixMilli(),
			Open:     b.Open,
			High:     b.High,
			Low:      b.Low,
			Close:    b.Close,
		}
	}
	return klines, nil
}

// GetCurrentPrice returns the latest trade price from Alpaca.
func (c *AlpacaClient) GetCurrentPrice() float64 {
	url := fmt.Sprintf("%s/stocks/trades/latest?symbols=%s", alpacaBaseURL, c.symbol)

	body, err := c.doRequest(url)
	if err != nil {
		return 0
	}

	var resp struct {
		Trades map[string]struct {
			Price float64 `json:"p"`
		} `json:"trades"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return 0
	}

	trade, ok := resp.Trades[c.symbol]
	if !ok {
		return 0
	}
	return trade.Price
}

// doRequest executes an authenticated GET request to Alpaca.
func (c *AlpacaClient) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("APCA-API-KEY-ID", c.apiKey)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

// mapIntervalToAlpaca converts short interval strings to Alpaca's format.
// Binance-style: "1m", "5m", "15m", "1h", "4h", "1d"
// Alpaca-style:  "1Min", "5Min", "15Min", "1Hour", "4Hour", "1Day"
func mapIntervalToAlpaca(interval string) string {
	mapping := map[string]string{
		"1m":  "1Min",
		"5m":  "5Min",
		"15m": "15Min",
		"30m": "30Min",
		"1h":  "1Hour",
		"2h":  "2Hour",
		"4h":  "4Hour",
		"1d":  "1Day",
		"1w":  "1Week",
	}
	if v, ok := mapping[strings.ToLower(interval)]; ok {
		return v
	}
	// Fallback: return as-is (user may have passed Alpaca format directly)
	return interval
}
