package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/annasblackhat/trading-alert/internal/telemetry"
)

// Server represents the API HTTP server.
type Server struct {
	port        string
	logProvider telemetry.LogProvider
}

// NewServer creates a new Server instance.
func NewServer(port string, logProvider telemetry.LogProvider) *Server {
	return &Server{
		port:        port,
		logProvider: logProvider,
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// GET /logs?lines=N to fetch recent N logs
	mux.HandleFunc("/logs", s.handleGetLogs)

	slog.Info("Starting HTTP server", slog.String("port", s.port))
	return http.ListenAndServe(":"+s.port, mux)
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Lift CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default to last 100 lines if lines query param is missing
	linesCount := 100
	linesParam := r.URL.Query().Get("lines")
	if linesParam != "" {
		parsed, err := strconv.Atoi(linesParam)
		if err == nil && parsed > 0 {
			// Cap at 1000 lines for safety
			if parsed > 1000 {
				linesCount = 1000
			} else {
				linesCount = parsed
			}
		}
	}

	// Fetch logs
	logs, err := s.logProvider.GetRecentLogs(linesCount)
	if err != nil {
		slog.Error("Failed to get recent logs", slog.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Construct JSON payload using the extracted lines
	// Since lines are pre-formatted JSON from slog, we'll return an array of strings
	// Or we can just stream plain text with proper content type. Plain text might be easier to read.
	// But let's return JSON to make it programmable.
	response := struct {
		Lines []string `json:"lines"`
		Total int      `json:"total"`
	}{
		Lines: logs,
		Total: len(logs),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to write logs response", slog.Any("error", err))
		return
	}
}
