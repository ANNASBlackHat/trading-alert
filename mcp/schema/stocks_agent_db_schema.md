# Database Schema — `stocks_agent` (MongoDB)

> Source of truth: `bots/ai_agent_stocks.py`
>
> - Database name: `stocks_agent`
> - Connection: `MONGODB_URI` env var → `pymongo.MongoClient`
> - Schema version: `v2` (bump when the extraction format changes; old and new docs coexist)
> - All dates/timestamps stored as ISO 8601 strings (UTC)
> - All upserts use `update_one(..., upsert=True)`; bulk card writes are delete-then-insert per `video_id`

---

## 1. `channels`

Static reference of tracked YouTube channels. Seeded from the `CHANNELS` config at startup.

| Field        | Type | Notes |
|--------------|------|-------|
| `channel_id` | str  | **Unique key** — upsert filter |
| `name`       | str  | Channel display name |

---

## 2. `processed_videos`

Tracks ingestion state. **Upsert key: `video_id` + `schema_version`.**

| Field          | Type      | Notes |
|----------------|-----------|-------|
| `video_id`     | str       | Part of upsert key |
| `schema_version` | str     | `"v2"` — part of upsert key; old docs remain queryable |
| `channel_id`   | str       | |
| `publish_date` | str       | `YYYY-MM-DD` |
| `processed_at` | str       | ISO timestamp (full datetime, used with `$regex` prefix match as a fallback filter) |
| `status`       | str       | `"success"` when complete |
| `format`       | str \| null | Video format/style tag |
| `content_style`| str \| null | |

Videos are skipped when a `{"video_id", "schema_version", "status": "success"}` doc already exists.

---

## 3. `speakers`

One doc per speaker identified in a video. **Rewritten per video** (`delete_many` + `insert_many` on `video_id`).

| Field         | Type      | Notes |
|---------------|-----------|-------|
| `video_id`    | str       | **Compound key** with video |
| `name`        | str       | |
| `role`        | str       | |
| `affiliation` | str \| null | |

---

## 4. `stock_cards`

One doc per stock mentioned. **Rewritten per video.** `card_id` auto-generated (UUID) if missing.

| Field                            | Type      | Notes |
|----------------------------------|-----------|-------|
| `card_id`                        | str       | UUID, unique per card |
| `video_id`                       | str       | |
| `schema_version`                 | str       | `"v2"` |
| `ticker`                         | str \| null | Resolved via `ticker_lookup`; set to null if unresolvable |
| `ticker_unresolved`              | bool      | Added by post-processing when the ticker couldn't be resolved |
| `company_name`                   | str       | |
| `sector`                         | str       | |
| `speaker_name`                   | str       | Who made the call |
| `timestamp_in_video`             | str \| null | e.g. "00:12:30" |
| `depth`                          | str       | |
| `stance`                         | str       | |
| `confidence_language`            | str       | |
| `reasoning_type`                 | list[str] | |
| `reasoning_summary`              | str       | |
| `historical_price_narrative`     | str \| null | |
| `price_at_time_of_recording`     | str \| null | |
| `forward_price_target_or_level`  | str \| null | |
| `past_catalysts_referenced`      | list[str] | |
| `upcoming_catalysts`             | list[str] | |
| `timeframe`                      | str       | |
| `direct_quote`                   | str       | |

---

## 5. `concept_cards`

One doc per market concept/macro idea discussed. **Rewritten per video.**

| Field              | Type      | Notes |
|--------------------|-----------|-------|
| `card_id`          | str       | UUID |
| `video_id`         | str       | |
| `schema_version`   | str       | `"v2"` |
| `concept`          | str       | |
| `concept_category` | str       | |
| `explanation_summary` | str    | |
| `applies_to_sector` | str \| null | |
| `timestamp_in_video` | str \| null | |

---

## 6. `ticker_lookup`

Reference table for company-name → ticker resolution.

| Field                      | Type | Notes |
|-----------------------------|------|-------|
| `company_name_normalized`   | str  | **Unique key** — upsert filter (lowercased/normalized company name) |
| `ticker`                    | str  | Uppercased |
| `exchange`                  | str  | e.g. `NASDAQ`, `NYSE` |

Can be bulk-seeded from a CSV (`load_ticker_lookup_csv`); individual upserts happen when cards are processed.

---

## 7. `agent_memory`

**Singleton collection** — exactly one document, replaced wholesale via `replace_one` on `{}`.

| Field                 | Type    | Notes |
|-----------------------|---------|-------|
| `last_updated`        | str     | ISO timestamp |
| `market_narrative`    | str     | |
| `sectors_consensus`   | dict    | |
| `technique_insights`  | list[str] | |
| `channel_reliability` | dict    | |
| `open_predictions`    | list[dict] | |
| `agent_current_view`  | str     | |
| `agent_reflection`    | str     | |

> Note: unlike the BTC bot, `sectors_consensus` (dict) replaces `key_levels_consensus` (KeyLevels) — this makes sense given the multi-asset focus.

---

## 8. `agent_opinions`

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

> Note: no `reflection` field here (the BTC bot has one).

---

## Recommended indexes

| Collection | Index | Reason |
|------------|-------|--------|
| `channels`         | `{channel_id: 1}` unique | Upsert key |
| `processed_videos` | `{video_id: 1, schema_version: 1, status: 1}` unique | Upsert key + skip check |
| `processed_videos` | `{publish_date: 1}` | "Today's videos" queries |
| `speakers`         | `{video_id: 1}` | Delete-then-insert |
| `stock_cards`      | `{video_id: 1}` | Delete-then-insert + "cards from today" join |
| `concept_cards`    | `{video_id: 1}` | Same |
| `ticker_lookup`    | `{company_name_normalized: 1}` unique | Lookup key |
| `agent_opinions`   | `{opinion_date: 1}` unique | Upsert key |
