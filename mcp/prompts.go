package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterPrompts adds the four reusable workflow prompts. Each returns a
// user-role message describing the recommended tool-call sequence, so the
// chat agent has a proven recipe it can follow.
func RegisterPrompts(srv *mcp.Server) {
	srv.AddPrompt(&mcp.Prompt{
		Name:        "daily_briefing",
		Description: "Today's market view: both agent states, yesterday's new analyses, and expiring predictions.",
	}, dailyBriefing)

	srv.AddPrompt(&mcp.Prompt{
		Name:        "weekly_review",
		Description: "Weekly digest: accuracy scoreboard, narrative changes, and top stock themes.",
	}, weeklyReview)

	srv.AddPrompt(&mcp.Prompt{
		Name:        "ticker_deep_dive",
		Description: "All cards, conflicts, catalysts, and speaker credibility for one ticker.",
		Arguments: []*mcp.PromptArgument{
			{Name: "ticker", Description: "Ticker symbol, e.g. NVDA", Required: true},
		},
	}, tickerDeepDive)

	srv.AddPrompt(&mcp.Prompt{
		Name:        "channel_report",
		Description: "Track record, technique usage, and reliability notes for one channel.",
		Arguments: []*mcp.PromptArgument{
			{Name: "channel", Description: "Channel name, e.g. 'Channel A'", Required: true},
		},
	}, channelReport)
}

func promptResult(text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: text},
			},
		},
	}
}

func dailyBriefing(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptResult(
		"Produce today's market briefing. " +
			"Steps: " +
			"1. Call btc_get_agent_state and stocks_get_agent_state for both rolling views. " +
			"2. Call btc_list_predictions with status=pending to surface expiring predictions. " +
			"3. Call get_pipeline_status to confirm today's ingestion completed. " +
			"Synthesize: the agent's current BTC narrative + open predictions with live prices, " +
			"the stocks sectors consensus, and any unresolved-ticker caveats. " +
			"End with a short 'watchlist' of the nearest target dates."), nil
}

func weeklyReview(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return promptResult(
		"Produce a weekly review. " +
			"Steps: " +
			"1. Call btc_list_predictions with status=scored, days=7 for last week's outcomes. " +
			"2. Call get_pipeline_status with since_days=7. " +
			"3. Call stocks_get_agent_state and btc_get_agent_state to note narrative drift. " +
			"Summarize: which techniques/channels were right, which were wrong, the top 3 stock " +
			"themes of the week, and whether the agent's consensus shifted."), nil
}

func tickerDeepDive(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	ticker := arg(req, "ticker")
	return promptResult(
		fmt.Sprintf(
			"Deep-dive ticker %s. Steps: 1. Call stocks_get_ticker_view with ticker=%q, since_days=30, include_quotes=true. "+
				"2. Call get_video_detail on the top 2-3 video_ids to recover speaker + full card context. "+
				"3. Reconcile the stance_breakdown against speaker credibility. "+
				"Deliver: who is bullish, who is bearish, their catalysts, the direct quotes, and any conflicting calls.",
			ticker, ticker)), nil
}

func channelReport(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	ch := arg(req, "channel")
	return promptResult(
		fmt.Sprintf(
			"Produce a channel report for %q. Steps: 1. Call list_channels to confirm the channel is tracked. "+
				"2. Call btc_list_predictions with days=90 to bucket by channel. "+
				"3. Pull the channel's reliability entry from btc_get_agent_state / stocks_get_agent_state. "+
				"Deliver: track record, most-used techniques, reliability notes, and open calls.",
			ch)), nil
}

// arg reads a prompt argument; returns "" when missing.
func arg(req *mcp.GetPromptRequest, key string) string {
	if req == nil || req.Params == nil {
		return ""
	}
	return req.Params.Arguments[key]
}
