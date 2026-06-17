package main

import (
	"bmc-rpc-service/bmc"
	"bmc-rpc-service/rpc"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: false,
		Error:   msg,
	})
}

func main() {
	sim := bmc.NewBMCSimulator(serverCount)
	sim.Start(tickInterval)

	if err := rpc.RegisterService(sim); err != nil {
		log.Fatalf("Failed to register RPC service: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "GET" {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		servers := sim.ListAll()
		writeJSON(w, http.StatusOK, servers)
	})

	mux.HandleFunc("/api/server/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		id := r.URL.Path[len("/api/server/"):]
		if id == "" {
			writeError(w, http.StatusBadRequest, "Server ID is required")
			return
		}

		switch r.Method {
		case "GET":
			status, ok := sim.GetStatus(id)
			if !ok {
				writeError(w, http.StatusNotFound, "Server not found")
				return
			}
			writeJSON(w, http.StatusOK, status)

		case "POST":
			var req struct {
				PowerLimit float64 `json:"power_limit"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}
			ok := sim.SetPowerLimit(id, req.PowerLimit)
			if !ok {
				writeError(w, http.StatusNotFound, "Server not found")
				return
			}
			status, _ := sim.GetStatus(id)
			writeJSON(w, http.StatusOK, status)

		default:
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"servers": serverCount,
			"uptime":  time.Since(startTime).String(),
		})
	})

	fmt.Printf("BMC RPC Service starting on %s\n", listenAddr)
	fmt.Printf("Simulating %d blade servers\n", serverCount)
	fmt.Printf("API endpoints:\n")
	fmt.Printf("  GET  /api/servers       - List all servers\n")
	fmt.Printf("  GET  /api/server/:id    - Get server status\n")
	fmt.Printf("  POST /api/server/:id    - Set power limit\n")
	fmt.Printf("  GET  /api/health        - Health check\n")

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

var startTime = time.Now()
