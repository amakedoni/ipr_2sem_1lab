// Дополнение к examples/backend/main.go:
// Для добавления метрик Prometheus в Go-приложение установите библиотеку:
//   go get github.com/prometheus/client_golang/prometheus
//   go get github.com/prometheus/client_golang/prometheus/promhttp
//
// Добавьте в main.go:
//
//   import "github.com/prometheus/client_golang/prometheus/promhttp"
//
//   // В функции main():
//   http.Handle("/metrics", promhttp.Handler())
//
// Это автоматически предоставит стандартные Go runtime метрики по пути /metrics.
// Prometheus будет их собирать через ServiceMonitor (task3-monitoring.yaml).

// ---- Пример backend/main.go с метриками ----
// (Замените содержимое examples/backend/main.go на этот файл)

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_http_requests_total",
		Help: "Общее количество HTTP запросов к backend",
	}, []string{"method", "path", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "backend_http_request_duration_seconds",
		Help:    "Время обработки HTTP запросов",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
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

func instrument(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		duration := time.Since(start).Seconds()
		requestsTotal.WithLabelValues(r.Method, path, "200").Inc()
		requestDuration.WithLabelValues(r.Method, path).Observe(duration)
	}
}

func main() {
	port := getEnv("PORT", "5000")
	podName := getEnv("HOSTNAME", "unknown")

	logger.Info("Starting backend server",
		"port", port,
		"pod_name", podName,
		"app_env", getEnv("APP_ENV", "development"),
	)

	http.HandleFunc("/", instrument("/", rootHandler))
	http.HandleFunc("/api/info", instrument("/api/info", infoHandler))
	http.HandleFunc("/health", instrument("/health", healthHandler))
	http.Handle("/metrics", promhttp.Handler()) // Prometheus endpoint

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
			"/metrics":  "Prometheus metrics",
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
