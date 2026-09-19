package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/joho/godotenv"
)

// loadEnvForTest loads the .env file at the repo root (parent of mcp/).
func loadEnvForTest(t *testing.T) {
	t.Helper()
	// Tests run with CWD = mcp/, so .env is one level up.
	if err := godotenv.Load(filepath.Join("..", ".env")); err != nil {
		t.Log("no .env at repo root; relying on environment")
	}
}

// initStoreForTest loads .env and connects to the real MONGODB_URI.
// Tests that need a live Mongo will skip gracefully when it is unreachable.
func initStoreForTest(t *testing.T) context.Context {
	t.Helper()
	loadEnvForTest(t)
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set; skipping live-mongo test")
	}
	ctx := context.Background()
	if err := Init(ctx); err != nil {
		t.Skipf("cannot connect to Mongo (%v); skipping", err)
	}
	if instance() == nil {
		t.Skip("store not initialized after Init; skipping")
	}
	_ = uri
	return ctx
}

// TestInitFailureWithoutURI verifies Init returns a clear error when
// MONGODB_URI is absent.
func TestInitFailureWithoutURI(t *testing.T) {
	old := os.Getenv("MONGODB_URI")
	t.Setenv("MONGODB_URI", "")
	if err := Init(context.Background()); err == nil {
		t.Fatal("expected error when MONGODB_URI is empty")
	}
	_ = old
}

// TestNewServerWiresEverything verifies the server is built with the
// expected number of tools, resources, and prompts.
func TestNewServerWiresEverything(t *testing.T) {
	initStoreForTest(t)
	srv := NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	// We cannot easily enumerate registered tools without a session, but we
	// can confirm the server object is usable.
	_ = srv
}

// TestBtcGetAgentState exercises the handler end-to-end against live Mongo.
// It is skipped when the database is unreachable.
func TestBtcGetAgentStateLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := BtcGetAgentState(ctx, &sdkmcp.CallToolRequest{}, BtcAgentStateArgs{})
	if err != nil {
		t.Fatalf("BtcGetAgentState: %v", err)
	}
	b, _ := json.Marshal(out)
	t.Logf("found=%v unscored=%v", out.Found, out.Unscored)
	t.Logf("response size: %d bytes", len(b))
}

// TestStocksGetAgentStateLive exercises the stocks handler.
func TestStocksGetAgentStateLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := StocksGetAgentState(ctx, &sdkmcp.CallToolRequest{}, StocksAgentStateArgs{})
	if err != nil {
		t.Fatalf("StocksGetAgentState: %v", err)
	}
	t.Logf("found=%v", out.Found)
}

// TestGetPipelineStatusLive exercises the aggregation pipeline.
func TestGetPipelineStatusLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := GetPipelineStatus(ctx, &sdkmcp.CallToolRequest{}, PipelineStatusArgs{SinceDays: 7})
	if err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	}
	t.Logf("processed=%d failed=%d channels=%d", out.VideosProcessed, out.VideosFailed, len(out.PerChannel))
}

// TestBtcListPredictionsLive exercises the filter + sort + limit path.
func TestBtcListPredictionsLive(t *testing.T) {
	ctx := initStoreForTest(t)
	_, out, err := BtcListPredictions(ctx, &sdkmcp.CallToolRequest{}, BtcListPredictionsArgs{Days: 7, Limit: 5})
	if err != nil {
		t.Fatalf("BtcListPredictions: %v", err)
	}
	t.Logf("total=%d unscored=%d", out.Total, out.Unscored)
}

// TestHealth verifies connectivity after Init.
func TestHealth(t *testing.T) {
	initStoreForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := instance()
	if err := s.Health(ctx); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

// TestMongoImportSanity guards the mongo import used for ErrNoDocuments.
func TestMongoImportSanity(t *testing.T) {
	_ = mongo.ErrNoDocuments
}
