package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/annasblackhat/trading-alert/internal/model"
)

type BinanceClient struct {
	symbol string
}

func NewBinanceClient(symbol string) *BinanceClient {
	return &BinanceClient{symbol: symbol}
}

// Symbol returns the trading symbol (satisfies MarketClient interface).
func (c *BinanceClient) Symbol() string { return c.symbol }

func (c *BinanceClient) FetchKlines(interval string, limit int) ([]model.Kline, error) {
	url := fmt.Sprintf("https://data-api.binance.vision/api/v3/klines?symbol=%s&interval=%s&limit=%d", c.symbol, interval, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var raw [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	klines := make([]model.Kline, len(raw))
	for i, v := range raw {
		klines[i] = model.Kline{
			OpenTime: int64(v[0].(float64)),
			Open:     mustParseFloat(v[1]),
			High:     mustParseFloat(v[2]),
			Low:      mustParseFloat(v[3]),
			Close:    mustParseFloat(v[4]),
		}
	}
	return klines, nil
}

func (c *BinanceClient) GetCurrentPrice() float64 {
	url := "https://data-api.binance.vision/api/v3/ticker/price?symbol=" + c.symbol
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var data struct {
		Price string `json:"price"`
	}
	if json.NewDecoder(resp.Body).Decode(&data) != nil {
		return 0
	}
	p, _ := strconv.ParseFloat(data.Price, 64)
	return p
}

func mustParseFloat(v interface{}) float64 {
	s := fmt.Sprintf("%v", v)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
