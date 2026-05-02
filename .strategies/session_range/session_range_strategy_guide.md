# Session Range & London Sweep Strategy Guide

**Strategy Type:** ICT / Smart Money — Session-Based Liquidity Sweep Reversal  
**Markets:** ES1! (S&P 500 Futures), NQ1! (Nasdaq 100 Futures)  
**Timeframes:** 1H (context), 15M (entry)  
**Entry Window:** NY Open — 12:00–17:00 UTC  
**Style:** Counter-trend reversal after liquidity sweep

---

## Table of Contents

1. [Core Concept](#1-core-concept)
2. [The Three-Phase Framework](#2-the-three-phase-framework)
3. [The Indicator — Session Ranges & London Sweeps](#3-the-indicator)
4. [Reading the Sentiment Dashboard](#4-reading-the-sentiment-dashboard)
5. [Buy Setup (Long)](#5-buy-setup-long)
6. [Sell Setup (Short)](#6-sell-setup-short)
7. [Signal Quick Reference](#7-signal-quick-reference)
8. [Daily Execution Process](#8-daily-execution-process)
9. [Pre-Trade Checklist](#9-pre-trade-checklist)
10. [Risk Management Rules](#10-risk-management-rules)
11. [What to Avoid](#11-what-to-avoid)

---

## 1. Core Concept

This strategy is built on the ICT (Inner Circle Trader) framework. The foundational idea is that large institutional players — banks and hedge funds — deliberately engineer **liquidity raids** before moving price in the true direction.

They push price above or below obvious swing levels (the Asian session highs and lows) to trigger stop orders placed by retail traders. Once those stops are collected, institutions reverse price sharply in the opposite direction.

The strategy positions you to trade that reversal — not the initial sweep.

**The key insight:**  
The London session move is manipulation. The New York session move is the real direction.

---

## 2. The Three-Phase Framework

Every trading day is divided into three sessions, each with a distinct role.

### Phase 1 — Asian Range (Accumulation)

- **Time:** 00:00 – 06:00 UTC
- **What happens:** Price consolidates and forms a defined range with a clear high and low.
- **Why it matters:** The Asian high and low become liquidity pools. Buy stops accumulate above the high; sell stops accumulate below the low. These are the targets that London will hunt.
- **Key requirement:** The range must be **tight** (≤20 pips by default). A wide, messy Asian range does not produce a clean setup. If the range is wide, there is no trade for the day.

### Phase 2 — London Push (Manipulation)

- **Time:** 06:00 – 12:00 UTC
- **What happens:** London opens with momentum and breaks one side of the Asian range — either above the high or below the low — triggering the stop orders sitting there.
- **Why it matters:** The direction of this sweep tells you the daily bias. London taking the high means sell stops were cleared above — price should ultimately fall. London taking the low means buy stops were cleared below — price should ultimately rise.
- **Critical rule:** Do NOT trade the London push. This is the manipulation phase. You are only observing and recording which side was swept.

### Phase 3 — NY Reversal (Distribution)

- **Time:** 12:00 – 17:00 UTC
- **What happens:** New York opens and price reverses against the London direction. This is where you enter.
- **Why it matters:** After liquidity has been collected in London, institutions now distribute their positions in the opposite direction. This is the real move and your trading opportunity.

### Phase Summary

| Phase | Session | Time (UTC) | Role | Your Action |
|---|---|---|---|---|
| 1 | Asian Range | 00:00 – 06:00 | Accumulation | Identify range high/low |
| 2 | London Push | 06:00 – 12:00 | Manipulation | Watch for sweep, record bias |
| 3 | NY Reversal | 12:00 – 17:00 | Distribution | Enter reversal trade |

---

## 3. The Indicator

The custom TradingView Pine Script indicator is named **"Session Ranges & London Sweeps"** (overlay = true).

### What it plots on the chart

- **Purple box** — Asian session range (00:00–06:00 UTC)
- **Red box** — London session range (06:00–12:00 UTC)
- **Green box** — New York session range (12:00–17:00 UTC)
- **Dashed lines** — Asian session High and Low extended across the chart as key levels
- **TIGHT label** — Appears on the Asian box when the range qualifies (≤20 pips threshold)
- **Sweep signal** — Marks the candle where London crosses the Asian High or Low
- **Sentiment Dashboard** — A data panel in the chart corner (position configurable) showing real-time bias readings

### Configurable settings

| Setting | Default | Description |
|---|---|---|
| Asia Session | 0000–0600 | Timezone-aware session window |
| London Session | 0800–1200 | Timezone-aware session window |
| NY Session | 1300–1700 | Timezone-aware session window |
| Timezone | UTC | Adjust to your broker's timezone |
| Tight Threshold | 20.0 pips | Max range size to qualify as "tight" |
| EMA Length | 50 | Trend EMA period |
| RSI Length | 14 | Momentum RSI period |
| Dashboard Position | top_right | Configurable corner placement |

---

## 4. Reading the Sentiment Dashboard

The dashboard updates on every bar close and shows four fields. Read all four before making any trade decision.

### Field 1 — Trend (EMA)

Compares current price to the trend EMA (default 50-period).

| Reading | Meaning | Implication |
|---|---|---|
| Bullish | Price is above the EMA | Upward trend bias |
| Bearish | Price is below the EMA | Downward trend bias |

Use this to confirm that your trade direction aligns with the broader trend. Trades taken against the trend require reduced position size.

### Field 2 — Momentum (RSI)

Uses a 14-period RSI to classify current momentum.

| Reading | RSI Value | Meaning |
|---|---|---|
| Strong | RSI > 60 | Momentum is clearly directional |
| Neutral | RSI 40–60 | No clear momentum edge |
| Weak | RSI < 40 | Momentum is fading or reversing |

For a buy setup, you want to see Strong or recovering momentum. For a sell setup, you want Weak or falling momentum.

### Field 3 — London Bias (Most Important)

This is the core signal of the strategy. It tells you which side of the Asian range London swept.

| Reading | What happened | Trade direction |
|---|---|---|
| Bearish Sweep | London took the Asian High (buy stops swept) | Look for BUY at NY open |
| Bullish Sweep | London took the Asian Low (sell stops swept) | Look for SELL at NY open |
| No signal / blank | London did not clearly sweep either side | No trade — skip the day |

**Counter-intuitive naming note:** "Bearish Sweep" means London moved up (bearishly sweeping stops above the high). The expected NY move is then UP (bullish reversal). The name refers to what London did to the stops — not the direction you trade.

### Field 4 — Asia Range

| Reading | Meaning | Action |
|---|---|---|
| Tight | Range ≤ configured threshold (default 20 pips) | Setup qualifies — proceed |
| Normal | Range exceeds threshold | No trade — skip the day |

**This is a binary filter.** If Asia Range reads "Normal," close your charts and wait for tomorrow. There is no setup.

---

## 5. Buy Setup (Long)

**Condition:** London swept the Asian High (Dashboard shows "Bearish Sweep")

### Setup requirements

1. Asia Range = **Tight**
2. London broke **above** the Asian High during 06:00–12:00 UTC
3. Dashboard London Bias = **Bearish Sweep**
4. Dashboard Trend (EMA) = Bullish (preferred) or Neutral (acceptable)
5. Dashboard Momentum = Strong or recovering from Weak

### Entry execution

- Wait for NY open (12:00–13:00 UTC)
- Switch to the 15M chart
- Wait for the first clear bullish reversal candle — ideally a strong green candle, bullish engulfing, or displacement candle (large body, small wicks)
- **Entry trigger:** Buy on the break above the high of the confirmation candle, or on a pullback to the candle's open or 50% level (Fibonacci midpoint)

### Stop loss

- Below the NY session low (the lowest wick printed after NY open)
- OR below the London sweep wick low (the bottom of the spike that swept the Asian High)
- Use whichever is closer while still giving the trade room to breathe

### Targets

| Target | Level | Notes |
|---|---|---|
| TP1 (50% position) | Asian High | The swept level often acts as resistance-turned-support |
| TP2 (remaining) | Previous day high or HTF resistance | Higher timeframe key level |
| Trail stop | Move to breakeven at TP1 | Locks in a risk-free trade |

### Minimum reward-to-risk

2:1 required. If TP1 does not offer at least 2x the distance of your stop, do not take the trade.

---

## 6. Sell Setup (Short)

**Condition:** London swept the Asian Low (Dashboard shows "Bullish Sweep")

### Setup requirements

1. Asia Range = **Tight**
2. London broke **below** the Asian Low during 06:00–12:00 UTC
3. Dashboard London Bias = **Bullish Sweep**
4. Dashboard Trend (EMA) = Bearish (preferred) or Neutral (acceptable)
5. Dashboard Momentum = Weak or rolling over from Strong

### Entry execution

- Wait for NY open (12:00–13:00 UTC)
- Switch to the 15M chart
- Wait for the first clear bearish reversal candle — a strong red candle, bearish engulfing, or displacement candle
- **Entry trigger:** Sell on the break below the low of the confirmation candle, or on a pullback to the candle's open or 50% level

### Stop loss

- Above the NY session high (the highest wick printed after NY open)
- OR above the London sweep wick high (the top of the spike that swept the Asian Low)
- Use whichever is closer while still giving the trade room

### Targets

| Target | Level | Notes |
|---|---|---|
| TP1 (50% position) | Asian Low | The swept level often acts as support-turned-resistance |
| TP2 (remaining) | Previous day low or HTF support | Higher timeframe key level |
| Trail stop | Move to breakeven at TP1 | Locks in a risk-free trade |

### Minimum reward-to-risk

2:1 required. Same rule as the long setup — if the math does not work, skip the trade.

---

## 7. Signal Quick Reference

| Asia Range | London Bias | Trend (EMA) | Momentum | Decision |
|---|---|---|---|---|
| Tight | Bearish Sweep (took High) | Bullish | Strong | BUY — Full size, high conviction |
| Tight | Bearish Sweep (took High) | Neutral | Strong | BUY — Full size |
| Tight | Bearish Sweep (took High) | Bearish | Neutral | BUY — Reduced size |
| Tight | Bullish Sweep (took Low) | Bearish | Weak | SELL — Full size, high conviction |
| Tight | Bullish Sweep (took Low) | Neutral | Weak | SELL — Full size |
| Tight | Bullish Sweep (took Low) | Bullish | Neutral | SELL — Reduced size |
| Normal | Any | Any | Any | SKIP — No trade today |
| Tight | No sweep / blank | Any | Any | SKIP — Wait, no signal |
| Tight | Any | Any | Any (no NY reversal candle) | SKIP — No confirmation |

---

## 8. Daily Execution Process

Follow this sequence every trading day.

### Step 1 — Check Asia Range at 06:00 UTC

Open your chart as London is about to open. Look at the purple Asian box on the chart and check the dashboard.

- **If Asia Range = Tight:** Setup qualifies. Continue to Step 2.
- **If Asia Range = Normal:** No trade today. Close the chart.

### Step 2 — Monitor London (06:00–12:00 UTC) for the sweep

Watch whether London breaks above the Asian High or below the Asian Low.

- You are looking for a clear violation — a candle close or wick that breaks through the level
- The indicator will mark the sweep candle and update the London Bias field
- Do not enter any trades during this window
- If London does not sweep either side by 11:30 UTC, there is likely no clean setup

### Step 3 — Confirm the bias at NY Open (12:00 UTC)

Read all four dashboard fields at the NY open:

- Asia Range = Tight ✓
- London Bias = Bearish Sweep or Bullish Sweep ✓
- Trend and Momentum = aligned or acceptable ✓

If any critical field is missing, reduce size significantly or skip.

### Step 4 — Wait for the NY reversal confirmation candle (15M chart)

Switch to the 15M chart. Watch the first 1–3 candles after NY open.

- For a buy: wait for a strong green candle, bullish engulf, or price rejection of lows
- For a sell: wait for a strong red candle, bearish engulf, or price rejection of highs
- A large-bodied displacement candle with small wicks is the highest-confidence signal
- Do not enter on just a wick — wait for a candle body close confirming the reversal

### Step 5 — Set stop and targets before entering

Before clicking buy or sell, define:

- Exact entry price
- Exact stop loss level (in points/ticks)
- TP1 and TP2 levels
- Position size based on 1% account risk

### Step 6 — Manage and close by 17:00 UTC

- Take 50% off at TP1 and move stop to breakeven
- Let the remaining position run to TP2
- Close all remaining positions by 17:00 UTC
- The strategy does not hold positions overnight or through the London close

---

## 9. Pre-Trade Checklist

Before entering any trade, confirm all of the following. All 5 must be checked for a full-size position. 3–4 = reduced size. Fewer than 3 = no trade.

- [ ] **Asia Range is Tight** — Dashboard reads "Tight," not "Normal"
- [ ] **London clearly swept one side** — Unambiguous break above the Asian High or below the Asian Low, not just a touch
- [ ] **Dashboard London Bias is confirmed** — Field shows "Bearish Sweep" or "Bullish Sweep," not blank
- [ ] **NY reversal candle confirmed on 15M** — At least one strong rejection/reversal candle printed after NY open, before entry
- [ ] **No major economic news within 30 minutes** — FOMC, CPI, NFP, and similar high-impact releases override all technical setups — check the economic calendar before every session

---

## 10. Risk Management Rules

### Position sizing

- **Maximum risk per trade:** 1% of account balance
- Calculate lot/contract size so a stop-out = exactly 1% loss
- For ES and NQ futures, account for the point value per contract before sizing

### Reward-to-risk

- **Minimum R:R:** 2:1 on every trade
- If TP1 (first target) is not at least 2x the stop distance away, do not take the trade
- Aim for 3:1 when trade conditions are ideal

### Stop management

- **Never move your stop further away** once the trade is live
- Move stop to **breakeven** once the trade has moved 1R in your favor (i.e., price has moved the same distance as your stop)
- After TP1 is hit (50% off), trail the stop behind recent swing lows (for longs) or swing highs (for shorts)

### Session rules

- **Do not trade the London session** (06:00–12:00 UTC) — this is manipulation, not your signal
- **Do not trade the Asian session** — this is consolidation
- **All entries during NY session only** (12:00–17:00 UTC)
- **Close all positions by 17:00 UTC** — avoid holding through end-of-day illiquidity

### Daily limits

- Maximum 2 trade attempts per day
- If the first setup fails (stop hit), you may take one more with full confirmation
- Never take a third trade as revenge or to recover losses

---

## 11. What to Avoid

These are the most common mistakes with this strategy.

**Do not trade a wide Asian range.**  
If the Asian session produced a large, choppy range, the liquidity levels are not clean enough to produce a reliable sweep and reversal. The "Tight" filter exists for this reason — respect it without exception.

**Do not enter during the London session.**  
The London push is the manipulation phase. It is designed to look like a trending move. Entering with London will put you on the wrong side of the reversal. Your entry window is strictly NY open.

**Do not enter without a reversal candle.**  
Simply seeing the London Bias field update is not enough to enter. You need price confirmation on the 15M chart at NY open. A setup without a confirmation candle is just a hypothesis, not a trade.

**Do not ignore the economic calendar.**  
High-impact events (FOMC rate decisions, CPI prints, NFP reports) can produce violent, irrational moves that invalidate the session range logic entirely. Always check the calendar for the day before committing to a trade.

**Do not revenge trade after a stop-out.**  
If the NY reversal fails and your stop is hit, the setup has been invalidated. The London sweep may have been a genuine trend continuation rather than a manipulation. Accept the 1% loss and wait for tomorrow's opportunity.

**Do not hold positions overnight.**  
This is a same-session strategy. Positions held through the NY close are exposed to gap risk and overnight news events that the strategy is not designed to handle.

---

## Glossary

| Term | Definition |
|---|---|
| Asian Range | The high-low range formed during the 00:00–06:00 UTC session |
| Tight Range | An Asian range within the configured pip threshold (default 20 pips) |
| Liquidity Pool | Cluster of stop orders above a high or below a low |
| Sweep / Liquidity Raid | Price moving through a level specifically to trigger stops, then reversing |
| London Bias | The direction determined by which side of the Asian range London swept |
| NY Reversal | The counter-move at NY open, opposite to the London push direction |
| Displacement Candle | A large-bodied candle with minimal wicks, indicating strong institutional momentum |
| Bearish Sweep | London swept the Asian High (buy stops taken above) — bullish NY reversal expected |
| Bullish Sweep | London swept the Asian Low (sell stops taken below) — bearish NY reversal expected |
| R / 1R | One unit of risk — the distance from entry to stop loss |
| Breakeven | Moving the stop loss to the entry price, eliminating risk on the trade |
| HTF | Higher timeframe (e.g., 4H or Daily chart) |
| ICT | Inner Circle Trader — the trading methodology this strategy is based on |

---

*This document is intended for use with AI language models, automated systems, and trading education. It is not financial advice. All trading involves risk.*
