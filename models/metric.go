package models

import (
	"time"
)

type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	DeviceID  string    `json:"device_id"`
	CPUUsage  float64   `json:"cpu_usage"` // 0-100
	MemoryMB  float64   `json:"memory_mb"`
	RPS       int       `json:"rps"`
}
