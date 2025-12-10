package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-microservice/models"
	"go-microservice/services"
)

type MetricsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewMetricsHandler(analyticsService *services.AnalyticsService) *MetricsHandler {
	return &MetricsHandler{
		analyticsService: analyticsService,
	}
}

func (h *MetricsHandler) ReceiveMetrics(w http.ResponseWriter, r *http.Request) {
	var metric models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Устанавливаем timestamp если не предоставлен
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	// Валидация
	if metric.CPUUsage < 0 || metric.CPUUsage > 100 {
		http.Error(w, "CPU usage must be between 0 and 100", http.StatusBadRequest)
		return
	}

	// Обработка аналитики (асинхронно через goroutine)
	avgRPS, isAnomaly := h.analyticsService.AddMetric(metric)

	response := map[string]interface{}{
		"status":      "processed",
		"timestamp":   metric.Timestamp,
		"rolling_avg": avgRPS,
		"is_anomaly":  isAnomaly,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MetricsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	total, anomalies, rate, rollingAvg := h.analyticsService.GetStats()
	window := h.analyticsService.GetCurrentWindow()

	response := map[string]interface{}{
		"total_metrics":        total,
		"anomalies_detected":   anomalies,
		"anomaly_rate_percent": rate,
		"rolling_average_rps":  rollingAvg,
		"window_size":          len(window),
		"current_window":       window,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
