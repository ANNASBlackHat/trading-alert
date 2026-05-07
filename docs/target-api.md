# Target Price Alerts API

This document describes the new target alert endpoints for the trading bot.

The HTTP server runs on port `8080` by default.

## Create a Target

Request:

```bash
curl -X POST http://localhost:8080/api/targets \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTCUSDT",
    "target_price": 81500,
    "direction": "up",
    "bot_name": "BTC-1h/5m"
  }'
```

Response example (201 Created):

```json
{
  "id": "btcusdt-1716067659000000000",
  "bot_name": "BTC-1h/5m",
  "symbol": "BTCUSDT",
  "target_price": 81500,
  "direction": "up",
  "created_at": "2026-05-06T12:34:19Z",
  "last_state": "unknown"
}
```

Notes:
- `symbol` is required.
- `target_price` must be greater than 0.
- `direction` must be either `up` or `down`.
- `bot_name` is optional. If provided, the target will only apply to that bot.

## List Targets

### List all targets

```bash
curl http://localhost:8080/api/targets
```

Response example:

```json
[
  {
    "id": "btcusdt-1716067659000000000",
    "bot_name": "BTC-1h/5m",
    "symbol": "BTCUSDT",
    "target_price": 81500,
    "direction": "up",
    "created_at": "2026-05-06T12:34:19Z",
    "last_state": "below"
  }
]
```

### List targets by symbol

```bash
curl "http://localhost:8080/api/targets?symbol=BTCUSDT"
```

### List targets by bot name

```bash
curl "http://localhost:8080/api/targets?bot_name=BTC-1h/5m"
```

## Get Target by ID

Request:

```bash
curl http://localhost:8080/api/targets/btcusdt-1716067659000000000
```

Response example:

```json
{
  "id": "btcusdt-1716067659000000000",
  "bot_name": "BTC-1h/5m",
  "symbol": "BTCUSDT",
  "target_price": 81500,
  "direction": "up",
  "created_at": "2026-05-06T12:34:19Z",
  "last_state": "below"
}
```

## Delete Target

Request:

```bash
curl -X DELETE http://localhost:8080/api/targets/btcusdt-1716067659000000000
```

Response example:

- `204 No Content`

No body is returned on successful deletion.

## Behavior

- Targets are stored in memory and shared between the API server and all running bots.
- A target triggers only when the current price crosses from the opposite side of the threshold:
  - `direction: "up"` triggers when price moves from below the target to at/above it.
  - `direction: "down"` triggers when price moves from above the target to at/below it.
- After a target fires, it is deleted automatically.
