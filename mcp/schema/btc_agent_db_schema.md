# Database Schema — `btc_agent` (MongoDB)

> Source of truth: `bots/ai_agent_btc.py`
>
> - Database name: `btc_agent`
> - Connection: `MONGODB_URI` env var → `pymongo.MongoClient`
> - Schema version: `1.0`
> - All dates stored as ISO 8601 strings (`YYYY-MM-DD` or full ISO timestamps, UTC)
> - All write operations use `update_one(..., upsert=True)` — there are no bare inserts

---

## 1. `processed_videos`

Tracks which YouTube videos have already been ingested.

| Field          | Type   | Notes                                              |
|----------------|--------|----------------------------------------------------|
| `video_id`     | str    | **Unique key** — upsert filter                    |
| `channel_id`   | str    |                                                    |
| `channel_name` | str    |                                                    |
| `title`        | str    |                                                    |
| `published_at` | str    | ISO timestamp                                       |
| `processed_at` | str    | Set by bot on completion, ISO timestamp             |
| `status`       | str    | `"done"` when successful                           |

---

## 2. `daily_analyses`

One document per video — the LLM's BTC market analysis.

| Field                     | Type                     | Notes |
|---------------------------|--------------------------|-------|
| `video_id`                | str                      | **Upsert key** |
| `channel_id`              | str                      |       |
| `channel_name`            | str                      |       |
| `analysis_date`           | str                      | `YYYY-MM-DD` |
| `schema_version`          | str                      | `"1.0"` |
| `btc_price_mentioned`     | float \| null            |       |
| `market_structure`        | str \| null              | `bullish` \| `bearish` \| `ranging` \| `unclear` |
| `techniques_used`         | list of [Technique](#technique-embedded-object) | |
| `predictions`             | [Predictions](#predictions-embedded-object)      | |
| `key_levels`              | [KeyLevels](#keylevels-embedded-object)          | |
| `catalysts`               | list[str]                |       |
| `contrarian_view`         | str \| null              |       |
| `raw_transcription`       | str \| null              | Stored only if under 8 000 chars |

### Technique (embedded object)

| Field       | Type      | Notes |
|-------------|-----------|-------|
| `name`      | str       | e.g. "RSI divergence" |
| `timeframe` | str \| null | e.g. "4H", "1D" |
| `signal`    | str       | `bullish` \| `bearish` \| `neutral` |

### Predictions (embedded object)

| Field       | Type                                    | Notes |
|-------------|-----------------------------------------|-------|
| `primary`   | dict \| null                            | `{scenario, direction (up/down/sideways), target, timeframe, confidence (high/medium/low), invalidation}` |
| `scenarios` | list of [ScenarioPrediction](#scenarioprediction-embedded-object) | |
| `short_term`| str \| null                             | Free text, next 1–3 days |
| `long_term` | str \| null                             | Free text, weeks/months |
| `narrative` | str \| null                             | Catch-all |

### ScenarioPrediction (embedded object)

| Field        | Type      | Notes |
|--------------|-----------|-------|
| `label`      | str       | `bull case` \| `bear case` \| `base case` |
| `condition`  | str       | e.g. "if BTC holds $98k" |
| `target`     | float \| null | |
| `probability`| str \| null | e.g. "60%" |

### KeyLevels (embedded object)

| Field        | Type       |
|--------------|------------|
| `support`    | list[float] |
| `resistance` | list[float] |

---

## 3. `predictions`

Flattened, scoreable prediction records — one doc per primary prediction.
**Upsert key: `video_id` + `target_date`.**

| Field            | Type      | Notes |
|------------------|-----------|-------|
| `video_id`       | str       | Part of upsert key |
| `channel_name`   | str       |       |
| `prediction_date`| str       | `YYYY-MM-DD` — when the video was published |
| `direction`      | str       | `up` \| `down` \| `sideways` |
| `target`         | float     | Price target |
| `timeframe`      | str       | e.g. "7-14 days" |
| `target_date`    | str       | `YYYY-MM-DD` — when to score; part of upsert key |
| `confidence`     | str       | `high` \| `medium` \| `low` |
| `invalidation`   | str \| null | e.g. "daily close below $96k" |
| `actual_price`   | float \| null | Filled in by price-scoring step |
| `outcome`        | str \| null | `correct` \| `partial` \| `wrong` \| `invalidated` (null = unscored) |
| `accuracy_score` | float \| null | 0.0–1.0 |

Pending predictions are queried as `{"target_date": {"$lte": today}, "outcome": None}`.

---

## 4. `technique_ledger`

Running ledger of technique performance. **One doc per technique name** (upsert key: `technique_name`).

| Field                 | Type       | Notes |
|-----------------------|------------|-------|
| `technique_name`      | str        | **Upsert key** |
| `description`         | str \| null | |
| `times_used`          | int        | |
| `correct_calls`       | int        | |
| `hit_rate`            | float      | `correct_calls / times_used` |
| `best_market_condition` | str \| null | Condition where technique performed best |
| `recent_examples`     | list[dict] | Last N entries: `{date, correct, condition, note}` |
| `last_updated`        | str        | ISO timestamp |

---

## 5. `agent_memory`

**Singleton collection** — the bot maintains exactly one document (replaced wholesale via `replace_one`).

| Field               | Type   | Notes |
|---------------------|--------|-------|
| `last_updated`      | str    | ISO timestamp |
| `market_narrative`  | str    | |
| `key_levels_consensus` | [KeyLevels](#keylevels-embedded-object) | |
| `technique_insights` | list[str] | |
| `channel_reliability` | dict | `{channel_name: {accuracy: float, note: str}}` |
| `open_predictions`  | list[dict] | Capped at `MAX_OPEN_PREDICTIONS` |
| `agent_current_view`| str    | |
| `agent_reflection`  | str    | |

---

## 6. `agent_opinions`

The agent's daily consolidated opinion. **Upsert key: `opinion_date`** (one doc per day).

| Field             | Type       | Notes |
|-------------------|------------|-------|
| `opinion_date`    | str        | `YYYY-MM-DD`, **upsert key** |
| `direction`       | str        | `up` \| `down` \| `sideways` |
| `price_target`    | float \| null | |
| `reasoning`       | str        | |
| `techniques_cited`| list[str]  | |
| `actual_price`    | float \| null | Filled later by price check |
| `outcome`         | str \| null | `correct` \| `wrong` \| … (null = not yet evaluated) |
| `reflection`      | str \| null | |

---

## Recommended indexes

| Collection          | Index | Reason |
|---------------------|-------|--------|
| `processed_videos`  | `{video_id: 1}` unique | Upsert key |
| `daily_analyses`    | `{video_id: 1}` unique | Upsert key |
| `daily_analyses`    | `{analysis_date: -1}` | Date-range queries |
| `predictions`       | `{video_id: 1, target_date: 1}` unique | Composite upsert key |
| `predictions`       | `{target_date: 1, outcome: 1}` | Pending-prediction queries |
| `technique_ledger`  | `{technique_name: 1}` unique | Upsert key |
| `agent_opinions`    | `{opinion_date: 1}` unique | Upsert key |
