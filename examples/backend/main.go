package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"
)

type InfoResponse struct {
	Service   string  `json:"service"`
	Message   string  `json:"message"`
	Timestamp string  `json:"timestamp"`
	PodName   string  `json:"pod_name"`
	Platform  string  `json:"platform"`
	GoVersion string  `json:"go_version"`
	Uptime    float64 `json:"uptime"`
	AppEnv    string  `json:"app_env"`
}

type HealthResponse struct {
	Status  string  `json:"status"`
	Service string  `json:"service"`
	Uptime  float64 `json:"uptime"`
}

type RootResponse struct {
	Message   string            `json:"message"`
	Endpoints map[string]string `json:"endpoints"`
}

var (
	startTime time.Time
	logger    *slog.Logger
)

func init() {
	startTime = time.Now()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func main() {
	port := getEnv("PORT", "5000")
	podName := getEnv("HOSTNAME", "unknown")

	logger.Info("Starting backend server",
		"port", port,
		"pod_name", podName,
		"app_env", getEnv("APP_ENV", "development"),
	)

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/api/info", infoHandler)
	http.HandleFunc("/health", healthHandler)

	logger.Info("Server started successfully", "address", fmt.Sprintf(":%s", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Root endpoint called", "method", r.Method, "path", r.URL.Path)
	response := RootResponse{
		Message: "Kubernetes Backend API",
		Endpoints: map[string]string{
			"/api/info": "Get system information",
			"/health":   "Health check",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Info endpoint called", "method", r.Method, "path", r.URL.Path)
	response := InfoResponse{
		Service:   "backend",
		Message:   "Hello from Kubernetes Backend!",
		Timestamp: time.Now().Format(time.RFC3339),
		PodName:   getEnv("HOSTNAME", "unknown"),
		Platform:  runtime.GOOS,
		GoVersion: runtime.Version(),
		Uptime:    time.Since(startTime).Seconds(),
		AppEnv:    getEnv("APP_ENV", "development"),
	}
	respondJSON(w, http.StatusOK, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "healthy",
		Service: "backend",
		Uptime:  time.Since(startTime).Seconds(),
	}
	respondJSON(w, http.StatusOK, response)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("Failed to encode JSON response", "error", err)
	}
}
