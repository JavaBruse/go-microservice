package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go-microservice/metrics"
	"go-microservice/models"
)

type AnalyticsService struct {
	mu               sync.RWMutex
	redisClient      *redis.Client
	metricsWindow    []models.Metric
	maxWindowSize    int
	anomalyCount     int
	totalProcessed   int
	anomalyThreshold float64
	metricsChan      chan models.Metric
	processingDone   chan bool
	rollingAvg       float64
	cacheTTL         time.Duration
}

func NewAnalyticsService(windowSize int, redisClient *redis.Client) *AnalyticsService {
	service := &AnalyticsService{
		redisClient:      redisClient,
		metricsWindow:    make([]models.Metric, 0, windowSize),
		maxWindowSize:    windowSize,
		anomalyThreshold: 2.0,
		metricsChan:      make(chan models.Metric, 1000),
		processingDone:   make(chan bool),
		rollingAvg:       0,
		cacheTTL:         5 * time.Minute,
	}

	service.restoreFromCache()
	go service.processMetrics()
	go service.periodicCacheSave()

	return service
}

func (s *AnalyticsService) restoreFromCache() {
	if s.redisClient == nil {
		return
	}

	ctx := context.Background()

	if stats, err := s.redisClient.Get(ctx, "analytics:stats").Result(); err == nil {
		var statsData struct {
			AnomalyCount   int `json:"anomaly_count"`
			TotalProcessed int `json:"total_processed"`
		}
		if json.Unmarshal([]byte(stats), &statsData) == nil {
			s.anomalyCount = statsData.AnomalyCount
			s.totalProcessed = statsData.TotalProcessed
		}
	}

	if windowData, err := s.redisClient.Get(ctx, "analytics:window").Result(); err == nil {
		var window []models.Metric
		if json.Unmarshal([]byte(windowData), &window) == nil {
			s.metricsWindow = window
			if len(window) > 0 {
				s.rollingAvg = s.calculateRollingAverage()
			}
		}
	}
}

func (s *AnalyticsService) periodicCacheSave() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.saveToCache()
	}
}

func (s *AnalyticsService) saveToCache() {
	if s.redisClient == nil {
		return
	}

	ctx := context.Background()
	s.mu.RLock()

	statsData := struct {
		AnomalyCount   int `json:"anomaly_count"`
		TotalProcessed int `json:"total_processed"`
	}{
		AnomalyCount:   s.anomalyCount,
		TotalProcessed: s.totalProcessed,
	}

	if statsJSON, err := json.Marshal(statsData); err == nil {
		s.redisClient.Set(ctx, "analytics:stats", statsJSON, s.cacheTTL)
	}

	if windowJSON, err := json.Marshal(s.metricsWindow); err == nil {
		s.redisClient.Set(ctx, "analytics:window", windowJSON, s.cacheTTL)
	}

	s.redisClient.Set(ctx, "analytics:rolling_avg", s.rollingAvg, s.cacheTTL)
	s.mu.RUnlock()
}

func (s *AnalyticsService) AddMetric(metric models.Metric) (float64, bool) {
	if s.redisClient != nil {
		ctx := context.Background()
		metricKey := fmt.Sprintf("metric:%s:%d", metric.DeviceID, metric.Timestamp.Unix())
		if metricJSON, err := json.Marshal(metric); err == nil {
			s.redisClient.Set(ctx, metricKey, metricJSON, s.cacheTTL)
		}
	}

	s.metricsChan <- metric

	s.mu.RLock()
	avg := s.rollingAvg
	s.mu.RUnlock()

	return avg, false
}

func (s *AnalyticsService) processMetrics() {
	for metric := range s.metricsChan {
		s.processSingleMetric(metric)
	}
	s.processingDone <- true
}

func (s *AnalyticsService) processSingleMetric(metric models.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalProcessed++
	metrics.IncrementMetricsProcessed()

	s.metricsWindow = append(s.metricsWindow, metric)
	if len(s.metricsWindow) > s.maxWindowSize {
		s.metricsWindow = s.metricsWindow[1:]
	}

	s.rollingAvg = s.calculateRollingAverage()

	if s.detectAnomaly(metric) {
		s.anomalyCount++
		metrics.IncrementAnomaliesDetected()

		if s.redisClient != nil {
			ctx := context.Background()
			anomalyKey := fmt.Sprintf("anomaly:%d", time.Now().UnixNano())
			anomalyData := map[string]interface{}{
				"metric":      metric,
				"timestamp":   time.Now(),
				"rolling_avg": s.rollingAvg,
			}
			if anomalyJSON, err := json.Marshal(anomalyData); err == nil {
				s.redisClient.Set(ctx, anomalyKey, anomalyJSON, s.cacheTTL)
			}
		}
	}

	metrics.SetRollingAverageRPS(s.rollingAvg)
	metrics.SetProcessingQueueSize(float64(len(s.metricsChan)))

	if s.redisClient != nil {
		go s.saveToCache()
	}
}

func (s *AnalyticsService) calculateRollingAverage() float64 {
	if len(s.metricsWindow) == 0 {
		return 0
	}

	sum := 0
	for _, m := range s.metricsWindow {
		sum += m.RPS
	}

	return float64(sum) / float64(len(s.metricsWindow))
}

func (s *AnalyticsService) detectAnomaly(metric models.Metric) bool {
	if len(s.metricsWindow) < 10 {
		return false
	}
	mean, stdDev := s.calculateStats()
	if stdDev == 0 {
		return false
	}
	zScore := math.Abs(float64(metric.RPS)-mean) / stdDev
	return zScore > s.anomalyThreshold
}

func (s *AnalyticsService) calculateStats() (float64, float64) {
	var sum float64
	var sqSum float64
	n := float64(len(s.metricsWindow))

	for _, m := range s.metricsWindow {
		sum += float64(m.RPS)
		sqSum += float64(m.RPS) * float64(m.RPS)
	}

	mean := sum / n
	variance := (sqSum / n) - (mean * mean)
	if variance < 0 {
		variance = 0
	}

	return mean, math.Sqrt(variance)
}

func (s *AnalyticsService) GetStats() (int, int, float64, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	anomalyRate := 0.0
	if s.totalProcessed > 0 {
		anomalyRate = float64(s.anomalyCount) / float64(s.totalProcessed) * 100
	}

	return s.totalProcessed, s.anomalyCount, anomalyRate, s.rollingAvg
}

func (s *AnalyticsService) GetCurrentWindow() []models.Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]models.Metric{}, s.metricsWindow...)
}

func (s *AnalyticsService) GetRecentAnomalies(limit int) ([]map[string]interface{}, error) {
	if s.redisClient == nil {
		return []map[string]interface{}{}, nil
	}

	ctx := context.Background()
	keys, err := s.redisClient.Keys(ctx, "anomaly:*").Result()
	if err != nil {
		return nil, err
	}

	var anomalies []map[string]interface{}
	for i, key := range keys {
		if i >= limit {
			break
		}
		if data, err := s.redisClient.Get(ctx, key).Result(); err == nil {
			var anomaly map[string]interface{}
			if json.Unmarshal([]byte(data), &anomaly) == nil {
				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies, nil
}

func (s *AnalyticsService) GetCacheStats() (int64, error) {
	if s.redisClient == nil {
		return 0, nil
	}

	ctx := context.Background()
	keys, err := s.redisClient.Keys(ctx, "*").Result()
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, key := range keys {
		if size, err := s.redisClient.MemoryUsage(ctx, key).Result(); err == nil {
			totalSize += size
		}
	}

	return totalSize, nil
}

func (s *AnalyticsService) Stop() {
	close(s.metricsChan)
	<-s.processingDone
	s.saveToCache()
}
