package api

import "github.com/annasblackhat/trading-alert/internal/model"

// MarketClient abstracts any exchange or data source.
// Implement this interface to add support for new exchanges (e.g., Bybit, OKX).
type MarketClient interface {
	FetchKlines(interval string, limit int) ([]model.Kline, error)
	GetCurrentPrice() float64
	Symbol() string
}
