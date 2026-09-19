package mcp

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ── get_pipeline_status ─────────────────────────────────────

// PipelineStatusArgs has no required fields.
type PipelineStatusArgs struct {
	SinceDays int `json:"since_days,omitempty" jsonschema:"window in days (default 7)"`
}

// PipelineStatus is the response for get_pipeline_status.
type PipelineStatus struct {
	WindowDays        int              `json:"window_days"`
	VideosProcessed   int              `json:"videos_processed"`
	VideosFailed      int              `json:"videos_failed"`
	PerChannel        []map[string]any `json:"per_channel"`
	UnresolvedTickers []string         `json:"unresolved_tickers,omitempty"`
	Disclaimer        string           `json:"disclaimer"`
}

// GetPipelineStatus reports ingestion counts, per-channel breakdown, and
// unresolved-ticker cards in the window.
func GetPipelineStatus(ctx context.Context, _ *sdkmcp.CallToolRequest, in PipelineStatusArgs) (*sdkmcp.CallToolResult, PipelineStatus, error) {
	s := instance()
	if s == nil {
		return nil, PipelineStatus{}, storeErr()
	}
	if in.SinceDays <= 0 {
		in.SinceDays = 7
	}
	since := time.Now().UTC().AddDate(0, 0, -in.SinceDays)
	sinceISO := since.Format("2006-01-02")

	out := PipelineStatus{WindowDays: in.SinceDays, Disclaimer: disclaimerText}

	// processed_videos in the window, grouped by status and channel.
	agg, err := s.stocks.Collection("processed_videos").Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"publish_date": bson.M{"$gte": sinceISO}}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"channel_id": "$channel_id", "status": "$status"},
			"count": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return nil, out, fmt.Errorf("aggregate processed_videos: %w", err)
	}
	defer agg.Close(ctx)

	var groups []struct {
		ID struct {
			ChannelID string `bson:"channel_id"`
			Status    string `bson:"status"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	if err := agg.All(ctx, &groups); err != nil {
		return nil, out, fmt.Errorf("decode processed_videos groups: %w", err)
	}

	byChannel := map[string]*map[string]any{}
	for _, g := range groups {
		if byChannel[g.ID.ChannelID] == nil {
			byChannel[g.ID.ChannelID] = &map[string]any{}
		}
		m := *byChannel[g.ID.ChannelID]
		switch g.ID.Status {
		case "success":
			m["success"] = g.Count
			out.VideosProcessed += g.Count
		case "failed":
			m["failed"] = g.Count
			out.VideosFailed += g.Count
		default:
			m["other"] = g.Count
		}
	}
	for ch, m := range byChannel {
		out.PerChannel = append(out.PerChannel, map[string]any{"channel_id": ch, "counts": m})
	}

	// Unresolved tickers in the window.
	uagg, err := s.stocks.Collection("stock_cards").Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"ticker_unresolved": true}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": "$ticker"}}},
	})
	if err == nil {
		defer uagg.Close(ctx)
		var groups []struct {
			Ticker string `bson:"_id"`
		}
		if err := uagg.All(ctx, &groups); err == nil {
			for _, g := range groups {
				if g.Ticker != "" {
					out.UnresolvedTickers = append(out.UnresolvedTickers, g.Ticker)
				}
			}
		}
	}

	return nil, out, nil
}

// ── get_video_detail ─────────────────────────────────────────

// VideoDetailArgs identifies one video.
type VideoDetailArgs struct {
	VideoID string `json:"video_id" jsonschema:"YouTube video ID"`
}

// VideoDetail joins every collection that references a single video.
type VideoDetail struct {
	Video        *ProcessedVideoStocks `json:"video,omitempty"`
	Speakers     []Speaker             `json:"speakers,omitempty"`
	StockCards   []StockCard           `json:"stock_cards,omitempty"`
	ConceptCards []ConceptCard         `json:"concept_cards,omitempty"`
	Disclaimer   string                `json:"disclaimer"`
}

// GetVideoDetail pulls every extraction for one video across all collections.
func GetVideoDetail(ctx context.Context, _ *sdkmcp.CallToolRequest, in VideoDetailArgs) (*sdkmcp.CallToolResult, VideoDetail, error) {
	s := instance()
	if s == nil {
		return nil, VideoDetail{}, storeErr()
	}
	out := VideoDetail{Disclaimer: disclaimerText}
	vid := in.VideoID

	var pv ProcessedVideoStocks
	if err := s.stocks.Collection("processed_videos").
		FindOne(ctx, bson.M{"video_id": vid}, options.FindOne()).Decode(&pv); err == nil {
		out.Video = &pv
	}

	cursor, err := s.stocks.Collection("speakers").
		Find(ctx, bson.M{"video_id": vid})
	if err == nil {
		defer cursor.Close(ctx)
		_ = cursor.All(ctx, &out.Speakers)
	}

	ccursor, err := s.stocks.Collection("stock_cards").
		Find(ctx, bson.M{"video_id": vid})
	if err == nil {
		defer ccursor.Close(ctx)
		var raw []map[string]any
		if ccursor.All(ctx, &raw) == nil {
			for _, m := range raw {
				out.StockCards = append(out.StockCards, hydrateStockCard(m))
			}
		}
	}

	kcursor, err := s.stocks.Collection("concept_cards").
		Find(ctx, bson.M{"video_id": vid})
	if err == nil {
		defer kcursor.Close(ctx)
		_ = kcursor.All(ctx, &out.ConceptCards)
	}

	return nil, out, nil
}

// ── list_channels ────────────────────────────────────────────

// ListChannelsArgs has no required fields.
type ListChannelsArgs struct{}

// ChannelsOut is the response for list_channels.
type ChannelsOut struct {
	Channels   []Channel `json:"channels"`
	Disclaimer string    `json:"disclaimer"`
}

// ListChannels returns the tracked channel reference table.
func ListChannels(ctx context.Context, _ *sdkmcp.CallToolRequest, _ ListChannelsArgs) (*sdkmcp.CallToolResult, ChannelsOut, error) {
	s := instance()
	if s == nil {
		return nil, ChannelsOut{}, storeErr()
	}
	cursor, err := s.stocks.Collection("channels").Find(ctx, bson.M{})
	if err != nil {
		return nil, ChannelsOut{}, fmt.Errorf("query channels: %w", err)
	}
	defer cursor.Close(ctx)

	var channels []Channel
	if err := cursor.All(ctx, &channels); err != nil {
		return nil, ChannelsOut{}, fmt.Errorf("decode channels: %w", err)
	}

	return nil, ChannelsOut{Channels: channels, Disclaimer: disclaimerText}, nil
}
