package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/annasblackhat/trading-alert/internal/target"
	"github.com/annasblackhat/trading-alert/internal/telemetry"
)

// Server represents the API HTTP server.
type Server struct {
	port        string
	logProvider telemetry.LogProvider
	targetStore target.Store
}

// NewServer creates a new Server instance.
func NewServer(port string, logProvider telemetry.LogProvider, targetStore target.Store) *Server {
	return &Server{
		port:        port,
		logProvider: logProvider,
		targetStore: targetStore,
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// GET /logs?lines=N to fetch recent N logs
	mux.HandleFunc("/logs", s.handleGetLogs)
	mux.HandleFunc("/api/targets", s.handleTargets)
	mux.HandleFunc("/api/targets/", s.handleTargetByID)

	slog.Info("Starting HTTP server", slog.String("port", s.port))
	return http.ListenAndServe(":"+s.port, mux)
}

type createTargetRequest struct {
	BotName     string  `json:"bot_name,omitempty"`
	Symbol      string  `json:"symbol"`
	TargetPrice float64 `json:"target_price"`
	Direction   string  `json:"direction"`
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		symbol := strings.ToUpper(r.URL.Query().Get("symbol"))
		botName := r.URL.Query().Get("bot_name")
		var targets []target.Target
		var err error

		if botName != "" {
			targets, err = s.targetStore.ListByBotName(botName)
		} else if symbol != "" {
			targets, err = s.targetStore.ListBySymbol(symbol)
		} else {
			targets, err = s.targetStore.ListAll()
		}

		if err != nil {
			slog.Error("Failed to list targets", slog.Any("error", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(targets); err != nil {
			slog.Error("Failed to write targets response", slog.Any("error", err))
		}

	case http.MethodPost:
		var req createTargetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Symbol == "" || req.TargetPrice <= 0 || !target.IsValidDirection(strings.ToLower(req.Direction)) {
			http.Error(w, "Missing or invalid target payload", http.StatusBadRequest)
			return
		}

		newTarget := target.Target{
			BotName:     req.BotName,
			Symbol:      strings.ToUpper(req.Symbol),
			TargetPrice: req.TargetPrice,
			Direction:   target.Direction(strings.ToLower(req.Direction)),
			CreatedAt:   time.Now().UTC(),
			LastState:   target.StateUnknown,
		}

		created, err := s.targetStore.Save(newTarget)
		if err != nil {
			slog.Error("Failed to save new target", slog.Any("error", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(created); err != nil {
			slog.Error("Failed to write created target response", slog.Any("error", err))
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTargetByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/targets/")
	if id == "" {
		http.Error(w, "Target ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		targetItem, err := s.targetStore.Get(id)
		if err != nil {
			http.Error(w, "Target not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(targetItem); err != nil {
			slog.Error("Failed to write target response", slog.Any("error", err))
		}

	case http.MethodDelete:
		if err := s.targetStore.Delete(id); err != nil {
			http.Error(w, "Target not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
