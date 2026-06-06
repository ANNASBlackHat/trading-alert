# Trading Alert Bot

A flexible, modular trading background bot built in Go using Clean Architecture. It actively polls the Binance data API to track market segments and alert users upon trading signals.

## Running the Bot

There are three ways to run this bot locally:

### 1. Standard Execution
Set the environment variables manually before running the execution point:
```bash
TELEGRAM_BOT_TOKEN="your_token" \
TELEGRAM_CHAT_ID="your_chat_id" \
go run cmd/bot/main.go
```

### 2. Using `.env` file
You can create a `.env` file in the root of the project:
```env
TELEGRAM_BOT_TOKEN=your_token
TELEGRAM_CHAT_ID=your_chat_id
```
And simply run the file:
```bash
go run cmd/bot/main.go
```

### 3. Using Docker
Build the docker image and run the minimal alpine container.
```bash
docker build -t trading-bot .
docker run --env-file .env trading-bot
```

> **Note**: The legacy `main.go` file is left in the root directory as a fallback. The modern version of the bot resides entirely within `cmd/` and `internal/`.

---

## Technical Guide: Extending the Bot

This project is built rigidly with interfaces meaning everything is "plug-and-play". You can easily add functionality without altering `bot.go` orchestrator code whatsoever.

### How to add a new Indicator

Indicators process market datasets and return a single declarative `TradeSignal` (which includes whether the trigger was met, and any stoploss/target zones).

1. Create a new struct and file in `internal/indicator/` (e.g. `rsi.go`):
```go
package indicator
import "github.com/annasblackhat/trading-alert/internal/model"

type RSI struct {
    Period int
    Threshold float64
}

func (r *RSI) Analyze(htfKlines []model.Kline, ltfKlines []model.Kline) model.TradeSignal {
    // -> Your math calculation here
    return model.TradeSignal{
       Trigger: true,
       EntryPrice: 50000,
    }
}
```

2. Open `cmd/bot/main.go`, instantiate your new indicator, and feed it into the `Bot` constructor:
```go
myRsi := indicator.RSI{Period: 14, Threshold: 30}
tradingBot := bot.NewBot(client, myRsi, telegramNotifier, htfInterval, ltfInterval)
```

### How to add a new Notifier

Notifiers are simple destinations for alert strings. To add something like Discord webhooks or Slack alerts:

1. Create a new structure in `internal/notifier/` (e.g. `discord.go`):
```go
package notifier

type DiscordNotifier struct {
    WebhookURL string
}

func (d *DiscordNotifier) Send(text string) error {
    // -> Perform HTTP Post to standard webhook URL here
    return nil
}
```

2. Open `cmd/bot/main.go` and swap out (or compose) the notifier:
```go
discordNotifier := notifier.DiscordNotifier{WebhookURL: "..."}
tradingBot := bot.NewBot(client, pitchfork, discordNotifier, htfInterval, ltfInterval)
```

> **Pro Tip:** In Go, you do not need to explicitly declare that a struct implements an `interface`. Simply adding the matching method signatures (`Analyze` or `Send`) is enough!

---

## API Documentation

The bot hosts an HTTP API on port `8080` to manage price targets and monitor log outputs.

### Endpoints

#### 1. Fetch Recent Logs
* **Route:** `GET /logs`
* **Query Params:** `lines=N` (optional, default `100`, max `1000`)
* **Example:**
  ```bash
  curl "http://localhost:8080/logs?lines=10"
  ```

#### 2. List Target Alerts
* **Route:** `GET /api/targets`
* **Query Params:** 
  - `symbol` (optional, e.g., `BTCUSDT`)
  - `bot_name` (optional, e.g., `BinanceBot`)
* **Example:**
  ```bash
  curl "http://localhost:8080/api/targets?symbol=BTCUSDT"
  ```

#### 3. Get Target Alert Details
* **Route:** `GET /api/targets/{id}`
* **Example:**
  ```bash
  curl "http://localhost:8080/api/targets/btcusdt-1780730646914702000"
  ```

#### 4. Delete Target Alert
* **Route:** `DELETE /api/targets/{id}`
* **Response:** `204 No Content` on success.
* **Example:**
  ```bash
  curl -X DELETE "http://localhost:8080/api/targets/btcusdt-1780730646914702000"
  ```

#### 5. Create Target Alert
* **Route:** `POST /api/targets`
* **Payload Fields:**
  - `bot_name` (string, optional): Restrict execution to a specific bot.
  - `symbol` (string, required): The ticker symbol (e.g. `"BTCUSDT"`).
  - `direction` (string, required): `"up"` (crossing/trailing rise) or `"down"` (crossing/trailing fall).
  - `type` (string, optional): `"price"` for standard alerts (default) or `"trailing"` for trailing stops.
  - `note` (string, optional): Context label included in the notification message.

##### Example A: Standard Price Alert (Type: `price`)
Triggers when price crosses the target price.
```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSDT",
    "direction": "up",
    "type": "price",
    "target_price": 68500.0,
    "note": "Take profit zone reached, sell 50%!"
  }'
```

##### Example B: Percentage Trailing Stop (Type: `trailing`)
Triggers when the price drops by 15% from its highest peak tracked *since setup*.
```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "ETHUSDT",
    "direction": "down",
    "type": "trailing",
    "trailing_percent": 15.0,
    "note": "Sell ETH, trailing stop hit!"
  }'
```

##### Example C: Absolute Value Trailing Stop (Type: `trailing`)
Triggers when the price rises by $50 from its lowest trough tracked *since setup*.
```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "SOLUSDT",
    "direction": "up",
    "type": "trailing",
    "trailing_value": 50.0,
    "note": "Reversal confirmed, buy entry trigger!"
  }'
```

##### Example D: Gated/Activated Trailing Stop (Type: `trailing`)
Triggers when the price drops 10% from the peak, but only activates *after* the price has first reached or exceeded `$70,000`.
```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSDT",
    "direction": "down",
    "type": "trailing",
    "trailing_percent": 10.0,
    "activation_price": 70000.0,
    "note": "Triggered trailing stop after reaching 70k!"
  }'
```