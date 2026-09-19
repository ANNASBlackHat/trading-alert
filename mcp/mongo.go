package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DbBtc is the database used by bots/ai_agent_btc.py.
	DbBtc = "btc_agent"
	// DbStocks is the database used by bots/ai_agent_stocks.py.
	DbStocks = "stocks_agent"
)

// disclaimerText is attached to every tool response so the LLM never
// presents the outputs as financial advice.
const disclaimerText = "Outputs are third-party analyst opinions and historical scores, not financial advice."

// Store holds two database handles sharing one Mongo client.
// Phase 1 is read-only.
type Store struct {
	client *mongo.Client
	btc    *mongo.Database
	stocks *mongo.Database
}

// store is the process-wide singleton, initialized in Init.
var store atomic.Pointer[Store]

// Init connects to both databases and stores the handle for the tool handlers.
// Must be called before the server starts accepting requests.
func Init(ctx context.Context) error {
	s, err := NewStore(ctx)
	if err != nil {
		return err
	}
	store.Store(s)
	return nil
}

// instance returns the initialized Store, or nil if Init failed. Callers
// (tool handlers) must check for nil and surface a clear error, since the
// server keeps running even when Mongo is unreachable.
func instance() *Store {
	return store.Load()
}

// storeErr returns a descriptive error when the store is unavailable.
func storeErr() error {
	return fmt.Errorf("mongodb not connected: is MONGODB_URI set and reachable? (the server stays up; retry later)")
}

// NewStore connects to Mongo using the MONGODB_URI environment variable,
// the same one the Python bots use. It pings the server to fail fast.
func NewStore(ctx context.Context) (*Store, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return nil, fmt.Errorf("MONGODB_URI is not set; add it to .env (loaded by cmd/mcp) or the environment")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	slog.Info("Connected to MongoDB",
		slog.String("databases", fmt.Sprintf("%s, %s", DbBtc, DbStocks)))

	return &Store{
		client: client,
		btc:    client.Database(DbBtc),
		stocks: client.Database(DbStocks),
	}, nil
}

// Close disconnects the underlying client.
func (s *Store) Close(ctx context.Context) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Disconnect(ctx)
}

// Health reports connectivity.
func (s *Store) Health(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.client.Ping(pingCtx, nil); err != nil {
		return fmt.Errorf("mongo ping: %w", err)
	}
	return nil
}
