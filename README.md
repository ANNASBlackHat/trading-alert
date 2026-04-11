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