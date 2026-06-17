package main

import (
	"bmc-rpc-service/bmc"
	"bmc-rpc-service/rpc"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

const (
	serverCount  = 40
	listenAddr   = ":8080"
	tickInterval = 2 * time.Second
)

type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type BatchPowerLimitRequest struct {
	ServerIDs  []string `json:"server_ids"`
	PowerLimit float64  `json:"power_limit"`
}

type BatchPowerLimitResponse struct {
	Updated []string           `json:"updated"`
	Failed  []string           `json:"failed"`
	Servers []bmc.ServerStatus `json:"servers"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(JSONResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(JSONResponse{
		Success: false,
		Error:   msg,
	}); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func panicRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC recovered: %v\nStack trace:\n%s", err, debug.Stack())
				writeError(w, http.StatusInternalServerError,
					fmt.Sprintf("Internal server error: %v", err))
			}
		}()
		next(w, r)
	}
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func main() {
	sim := bmc.NewBMCSimulator(serverCount)
	if sim == nil {
		log.Fatal("Failed to create BMC simulator")
	}
	sim.Start(tickInterval)

	if err := rpc.RegisterService(sim); err != nil {
		log.Fatalf("Failed to register RPC service: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/servers", panicRecovery(func(w http.ResponseWriter, r *http.Request) {
		if setCORSHeaders(w, r) {
			return
		}
		if r.Method != "GET" {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		if sim == nil {
			writeError(w, http.StatusInternalServerError, "Simulator not initialized")
			return
		}
		servers := sim.ListAll()
		writeJSON(w, http.StatusOK, servers)
	}))

	mux.HandleFunc("/api/server/", panicRecovery(func(w http.ResponseWriter, r *http.Request) {
		if setCORSHeaders(w, r) {
			return
		}

		if sim == nil {
			writeError(w, http.StatusInternalServerError, "Simulator not initialized")
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/api/server/")
		id = strings.TrimSpace(id)
		if id == "" {
			writeError(w, http.StatusBadRequest, "Server ID is required")
			return
		}

		switch r.Method {
		case "GET":
			status, ok := sim.GetStatus(id)
			if !ok {
				writeError(w, http.StatusNotFound, "Server not found: "+id)
				return
			}
			writeJSON(w, http.StatusOK, status)

		case "POST":
			var req struct {
				PowerLimit float64 `json:"power_limit"`
			}
			if r.Body == nil {
				writeError(w, http.StatusBadRequest, "Request body is required")
				return
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
				return
			}
			ok := sim.SetPowerLimit(id, req.PowerLimit)
			if !ok {
				writeError(w, http.StatusNotFound, "Server not found: "+id)
				return
			}
			status, _ := sim.GetStatus(id)
			writeJSON(w, http.StatusOK, status)

		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}))

	mux.HandleFunc("/api/servers/batch-power-limit", panicRecovery(func(w http.ResponseWriter, r *http.Request) {
		if setCORSHeaders(w, r) {
			return
		}

		if r.Method != "POST" {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		if sim == nil {
			writeError(w, http.StatusInternalServerError, "Simulator not initialized")
			return
		}

		if r.Body == nil {
			writeError(w, http.StatusBadRequest, "Request body is required")
			return
		}

		var req BatchPowerLimitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}

		if len(req.ServerIDs) == 0 {
			writeError(w, http.StatusBadRequest, "server_ids is required and must not be empty")
			return
		}

		if req.PowerLimit < 100 || req.PowerLimit > 1000 {
			writeError(w, http.StatusBadRequest, "power_limit must be between 100 and 1000")
			return
		}

		response := BatchPowerLimitResponse{
			Updated: make([]string, 0, len(req.ServerIDs)),
			Failed:  make([]string, 0),
			Servers: make([]bmc.ServerStatus, 0, len(req.ServerIDs)),
		}

		for _, id := range req.ServerIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if sim.SetPowerLimit(id, req.PowerLimit) {
				response.Updated = append(response.Updated, id)
				if status, ok := sim.GetStatus(id); ok {
					response.Servers = append(response.Servers, status)
				}
			} else {
				response.Failed = append(response.Failed, id)
			}
		}

		writeJSON(w, http.StatusOK, response)
	}))

	mux.HandleFunc("/api/health", panicRecovery(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"servers": serverCount,
			"uptime":  time.Since(startTime).String(),
		})
	}))

	fmt.Printf("BMC RPC Service starting on %s\n", listenAddr)
	fmt.Printf("Simulating %d blade servers\n", serverCount)
	fmt.Printf("API endpoints:\n")
	fmt.Printf("  GET  /api/servers                 - List all servers\n")
	fmt.Printf("  GET  /api/server/:id              - Get server status\n")
	fmt.Printf("  POST /api/server/:id              - Set power limit for single server\n")
	fmt.Printf("  POST /api/servers/batch-power-limit - Set power limit for multiple servers\n")
	fmt.Printf("  GET  /api/health                  - Health check\n")

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

var startTime = time.Now()
