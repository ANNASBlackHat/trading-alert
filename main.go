package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
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

type Kline struct {
	OpenTime int64
	Open     float64
	High     float64
	Low      float64
	Close    float64
}

type ActiveSignal struct {
	EntryPrice float64
	TPPrice    float64 // Red Magic Line (Take-Profit zone)
	SLPrice    float64
	Notified50 bool
	Notified70 bool // "30% closer"
	Notified90 bool // "10% closer"
	SignalTime time.Time
}

var (
	magicUpper     float64
	magicLower     float64
	active         *ActiveSignal
	lastCandleTime int64 // to fire trigger only once per new 5m candle
	tgToken        = os.Getenv("TELEGRAM_BOT_TOKEN")
	tgChatID       = os.Getenv("TELEGRAM_CHAT_ID")
)

func main() {
	fmt.Println("🚀 Pitchfork Strategy Bot started (BTCUSDT 5m) - Long Only")
	fmt.Println("   Polling every 15s | Telegram alerts enabled")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		runCycle()
	}
}

func runCycle() {
	// 1. Fetch HTF (1h) for Magic Lines
	htfKlines, err := fetchKlines(htfInterval, 500)
	if err != nil {
		fmt.Printf("❌ HTF fetch error: %v\n", err)
		return
	}

	// Compute latest confirmed pivots (exact Pine ta.pivothigh / ta.pivotlow logic)
	if ph, ok := latestPivotHigh(htfKlines, pivotLength, pivotLength); ok {
		magicUpper = ph
	}
	if pl, ok := latestPivotLow(htfKlines, pivotLength, pivotLength); ok {
		magicLower = pl
	}

	// 2. Fetch LTF (5m) for trigger logic
	ltfKlines, err := fetchKlines(ltfInterval, 100)
	if err != nil {
		fmt.Printf("❌ LTF fetch error: %v\n", err)
		return
	}
	if len(ltfKlines) < suckerCandles+2 {
		return
	}

	// 3. Current price for proximity alerts (real-time feel)
	currentPrice := getCurrentPrice()

	// 4. Check for new completed 5m candle
	latest := ltfKlines[len(ltfKlines)-1]
	if latest.OpenTime == lastCandleTime {
		// Same candle → only check proximity
		checkProximity(currentPrice)
		return
	}

	// New candle closed → process trigger
	lastCandleTime = latest.OpenTime

	// Compute Pitchfork trigger exactly as in Pine Script
	trigger, lowestDuringDrop, entryPrice, slPrice := checkPitchforkTrigger(ltfKlines)
	fmt.Println("trigger", trigger)
	fmt.Println("magicLower", magicLower)
	fmt.Println("lowestDuringDrop", lowestDuringDrop)

	if trigger && magicLower > 0 {
		fmt.Println("✅ PITCHFORK BUY SIGNAL DETECTED!")

		// Send Telegram alert
		msg := fmt.Sprintf(`🚨 PITCHFORK BUY ALERT!
Sucker move exhausted in 1H Liquidity Zone
Green 5m candle closed

Entry: Buy Stop above %.2f
SL: below %.2f
TP Zone: %.2f (Red Magic Line)

Signal time: %s`, entryPrice, slPrice, magicUpper, time.Now().Format(time.RFC3339))
		sendTelegram(msg)

		// Activate proximity monitoring
		active = &ActiveSignal{
			EntryPrice: entryPrice,
			TPPrice:    magicUpper,
			SLPrice:    slPrice,
			SignalTime: time.Now(),
		}
	}

	// 5. Always check proximity if we have an active signal
	checkProximity(currentPrice)
}

// ====================== CORE LOGIC (exact Pine translation) ======================

func checkPitchforkTrigger(klines []Kline) (bool, float64, float64, float64) {
	n := len(klines)
	current := klines[n-1] // just-closed 5m candle

	// isGreen = current candle must be green
	isGreen := current.Close > current.Open

	// Sucker Move: previous 'suckerCandles' must ALL be red
	isSuckerMove := true
	for i := n - 1 - suckerCandles; i < n-1; i++ {
		if klines[i].Close >= klines[i].Open {
			isSuckerMove = false
			break
		}
	}

	// lowest low during the sucker move (previous N candles)
	lowestDuringDrop := klines[n-1-1].Low
	for i := n - 1 - suckerCandles; i < n-1; i++ {
		if klines[i].Low < lowestDuringDrop {
			lowestDuringDrop = klines[i].Low
		}
	}

	// Zone touch check
	supZoneUpperBound := magicLower * (1 + zoneTolerance)
	touchedZone := lowestDuringDrop <= supZoneUpperBound && lowestDuringDrop > 0

	trigger := isGreen && isSuckerMove && touchedZone

	entryPrice := current.High
	slPrice := current.Low
	if lowestDuringDrop < slPrice {
		slPrice = lowestDuringDrop
	}

	return trigger, lowestDuringDrop, entryPrice, slPrice
}

// ====================== PIVOT FUNCTIONS (exact match to ta.pivothigh/ta.pivotlow) ======================

func latestPivotHigh(klines []Kline, left, right int) (float64, bool) {
	n := len(klines)
	if n < left+right+1 {
		return 0, false
	}
	for i := n - right - 1; i >= left; i-- {
		val := klines[i].High
		isPivot := true
		// left bars
		for j := 1; j <= left; j++ {
			if klines[i-j].High >= val {
				isPivot = false
				break
			}
		}
		if !isPivot {
			continue
		}
		// right bars
		for j := 1; j <= right; j++ {
			if klines[i+j].High >= val {
				isPivot = false
				break
			}
		}
		if isPivot {
			return val, true
		}
	}
	return 0, false
}

func latestPivotLow(klines []Kline, left, right int) (float64, bool) {
	n := len(klines)
	if n < left+right+1 {
		return 0, false
	}
	for i := n - right - 1; i >= left; i-- {
		val := klines[i].Low
		isPivot := true
		for j := 1; j <= left; j++ {
			if klines[i-j].Low <= val {
				isPivot = false
				break
			}
		}
		if !isPivot {
			continue
		}
		for j := 1; j <= right; j++ {
			if klines[i+j].Low <= val {
				isPivot = false
				break
			}
		}
		if isPivot {
			return val, true
		}
	}
	return 0, false
}

// ====================== API HELPERS ======================

func fetchKlines(interval string, limit int) ([]Kline, error) {
	url := fmt.Sprintf("https://data-api.binance.vision/api/v3/klines?symbol=%s&interval=%s&limit=%d", symbol, interval, limit)
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

	klines := make([]Kline, len(raw))
	for i, v := range raw {
		klines[i] = Kline{
			OpenTime: int64(v[0].(float64)),
			Open:     mustParseFloat(v[1]),
			High:     mustParseFloat(v[2]),
			Low:      mustParseFloat(v[3]),
			Close:    mustParseFloat(v[4]),
		}
	}
	return klines, nil
}

func getCurrentPrice() float64 {
	url := "https://data-api.binance.vision/api/v3/ticker/price?symbol=" + symbol
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

// ====================== TELEGRAM + PROXIMITY ======================

func sendTelegram(text string) {
	if tgToken == "" || tgChatID == "" {
		fmt.Printf("📨 TELEGRAM (console only): %s\n\n", text)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&parse_mode=HTML&text=%s",
		tgToken, tgChatID, url.QueryEscape(text))
	http.Get(url) // fire and forget
}

func checkProximity(current float64) {
	if active == nil || active.TPPrice <= active.EntryPrice {
		return
	}

	progress := (current - active.EntryPrice) / (active.TPPrice - active.EntryPrice)

	if progress >= 0.5 && !active.Notified50 {
		sendTelegram(fmt.Sprintf("📈 50%% to Red Line (TP)!\nPrice: %.2f | Progress: %.1f%%", current, progress*100))
		active.Notified50 = true
	}
	if progress >= 0.7 && !active.Notified70 {
		sendTelegram(fmt.Sprintf("📈 70%% to Red Line (30%% closer)!\nPrice: %.2f | Progress: %.1f%%", current, progress*100))
		active.Notified70 = true
	}
	if progress >= 0.9 && !active.Notified90 {
		sendTelegram(fmt.Sprintf("📈 90%% to Red Line (10%% closer)!\nPrice: %.2f | Progress: %.1f%%", current, progress*100))
		active.Notified90 = true
	}

	// Optional: reset when TP is reached
	if progress >= 1.0 {
		active = nil
	}
}
