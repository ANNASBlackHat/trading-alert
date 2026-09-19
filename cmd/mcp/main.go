// Command mcp runs the Trading Agent Memory MCP server.
//
// Usage:
//
//	MONGODB_URI=... go run ./cmd/mcp                       # stdio transport
//	MCP_TRANSPORT=http MCP_HTTP_ADDR=:8080 go run ./cmd/mcp # streamable HTTP
//
// A .env file in the working directory is loaded first, mirroring cmd/bot.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/annasblackhat/trading-alert/mcp"
)

func main() {
	// Load .env if present; silently fall through to system env when absent.
	if err := godotenv.Load(); err != nil {
		log.Print("note: no .env file; relying on environment variables")
	}

	// Connect to both databases up front so a bad URI fails fast.
	// We do NOT fatal on connect failure: the MCP server is stateless, so
	// even if Mongo is unreachable right now, the HTTP listener should
	// still come up — individual tool calls will surface the connection
	// error, and a retry can succeed once the network is available.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mcp.Init(ctx); err != nil {
		log.Printf("warning: mongo not reachable at startup: %v (server will start; tool calls may fail until connectivity is restored)", err)
	}

	srv := mcp.NewServer()

	if os.Getenv("MCP_TRANSPORT") == "http" {
		runHTTP(srv, cancel)
	} else {
		runStdio(srv, ctx, cancel)
	}
}

// runStdio serves over stdin/stdout, the default for local MCP clients.
// Logging goes to stderr so the JSON-RPC stream on stdout stays clean.
func runStdio(srv *sdkmcp.Server, ctx context.Context, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Print("shutting down")
		cancel()
	}()

	t := &sdkmcp.LoggingTransport{Transport: &sdkmcp.StdioTransport{}, Writer: os.Stderr}
	if err := srv.Run(ctx, t); err != nil {
		log.Fatalf("mcp server: %v", err)
	}
}

// runHTTP serves the streamable-HTTP transport in the foreground, blocking
// until SIGINT/SIGTERM.
func runHTTP(srv *sdkmcp.Server, cancel context.CancelFunc) {
	addr := os.Getenv("MCP_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return srv
	}, &sdkmcp.StreamableHTTPOptions{Stateless: true})
	httpSrv := &http.Server{Addr: addr, Handler: handler}

	go func() {
		log.Printf("MCP streamable-HTTP transport listening at %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server stopped: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Print("shutting down")
	if err := httpSrv.Close(); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	cancel()
}
