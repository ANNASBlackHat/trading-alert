package mcp

import (
	"context"
	_ "embed"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//go:embed schema/btc_agent_db_schema.md
var schemaBtc string

//go:embed schema/stocks_agent_db_schema.md
var schemaStocks string

// Resource URIs.
const (
	schemaBtcURI    = "schema://btc_agent"
	schemaStocksURI = "schema://stocks_agent"
	memoryBtcURI    = "memory://btc/current"
	memoryStocksURI = "memory://stocks/current"
)

// RegisterResources adds the four read-only resources to the server.
// Schema docs are embedded at build time (self-contained binary); memory
// resources are read live from Mongo on each fetch.
func RegisterResources(srv *sdkmcp.Server) {
	srv.AddResource(&sdkmcp.Resource{
		Name:        "schema://btc_agent",
		Description: "Database schema reference for the btc_agent MongoDB database.",
		URI:         schemaBtcURI,
		MIMEType:    "text/markdown",
	}, staticMarkdownResource(schemaBtcURI, schemaBtc))

	srv.AddResource(&sdkmcp.Resource{
		Name:        "schema://stocks_agent",
		Description: "Database schema reference for the stocks_agent MongoDB database.",
		URI:         schemaStocksURI,
		MIMEType:    "text/markdown",
	}, staticMarkdownResource(schemaStocksURI, schemaStocks))

	srv.AddResource(&sdkmcp.Resource{
		Name:        "memory://btc/current",
		Description: "Live snapshot of the btc_agent's rolling memory document.",
		URI:         memoryBtcURI,
		MIMEType:    "application/json",
	}, memoryBtcResource)

	srv.AddResource(&sdkmcp.Resource{
		Name:        "memory://stocks/current",
		Description: "Live snapshot of the stocks_agent's rolling memory document.",
		URI:         memoryStocksURI,
		MIMEType:    "application/json",
	}, memoryStocksResource)
}

func staticMarkdownResource(uri, text string) sdkmcp.ResourceHandler {
	return func(_ context.Context, _ *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{{
				URI:      uri,
				MIMEType: "text/markdown",
				Text:     text,
			}},
		}, nil
	}
}

func memoryBtcResource(ctx context.Context, _ *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	s := instance()
	if s == nil {
		return nil, storeErr()
	}
	var mem BtcAgentMemory
	err := s.btc.Collection("agent_memory").FindOne(ctx, bson.M{}, options.FindOne()).Decode(&mem)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}
	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      memoryBtcURI,
			MIMEType: "application/json",
			Text:     toJSON(mem),
		}},
	}, nil
}

func memoryStocksResource(ctx context.Context, _ *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	s := instance()
	if s == nil {
		return nil, storeErr()
	}
	var mem StocksAgentMemory
	err := s.stocks.Collection("agent_memory").FindOne(ctx, bson.M{}, options.FindOne()).Decode(&mem)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}
	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      memoryStocksURI,
			MIMEType: "application/json",
			Text:     toJSON(mem),
		}},
	}, nil
}

// toJSON marshals v to compact JSON; an empty object when v is the zero value.
func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
