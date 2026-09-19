package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// QuoteProvider gives live prices for arbitrary tickers so "any stock"
// alerts work, not just BTC. It reads FINNHUB_API_KEY from the environment
// (the same key the trading-bot already uses, see config.yaml finnhub.api_key).
type QuoteProvider struct {
	apiKey string
	client *http.Client
}

// NewQuoteProvider builds a provider from FINNHUB_API_KEY. Returns a working
// provider when the key is set, or a disabled provider (IsEnabled()==false)
// when it is not — callers degrade gracefully and report "no live quotes".
func NewQuoteProvider() *QuoteProvider {
	key := os.Getenv("FINNHUB_API_KEY")
	if key == "" {
		// Mirror cmd/bot: try .env in the working dir, then the parent
		// (tests run from mcp/).
		for _, p := range []string{".env", "../.env"} {
			if err := loadDotenvFile(p); err == nil {
				key = os.Getenv("FINNHUB_API_KEY")
				if key != "" {
					break
				}
			}
		}
	}
	if key == "" {
		return &QuoteProvider{client: &http.Client{Timeout: 10 * time.Second}}
	}
	return &QuoteProvider{apiKey: key, client: &http.Client{Timeout: 10 * time.Second}}
}

// IsEnabled reports whether live quotes can actually be fetched.
func (q *QuoteProvider) IsEnabled() bool { return q != nil && q.apiKey != "" }

// GetQuote returns the current price for a ticker via Finnhub. Works for US
// tickers (NVDA, SPY, TSLA) and many international symbols (005930.KS, AAPL).
// Returns 0 and an error when the symbol is unsupported or the key is unset.
func (q *QuoteProvider) GetQuote(ticker string) (float64, error) {
	if !q.IsEnabled() {
		return 0, fmt.Errorf("FINNHUB_API_KEY not set; live stock quotes unavailable")
	}
	sym := NormalizeSymbol(ticker)
	url := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", sym, q.apiKey)
	resp, err := q.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("finnhub quote: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("finnhub HTTP %d for %s", resp.StatusCode, sym)
	}
	var out struct {
		C float64 `json:"c"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("finnhub decode: %w", err)
	}
	if out.C == 0 {
		return 0, fmt.Errorf("finnhub returned no price for %s (unsupported symbol or market closed)", sym)
	}
	return out.C, nil
}

// NormalizeSymbol uppercases and trims a user-supplied ticker.
func NormalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// loadDotenvFile is a dependency-free .env loader (subset of godotenv): it
// reads KEY=VALUE lines and sets them in the environment when not already
// set. Good enough for picking up FINNHUB_API_KEY during tests.
func loadDotenvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
	return nil
}
