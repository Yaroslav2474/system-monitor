package models

import "time"

// MonitorData - основная структура для передачи данных
type MonitorData struct {
	CPULoad      float64   `json:"cpu_load"`
	GPULoad      float64   `json:"gpu_load"`
	TopProcesses []Process `json:"top_processes"`
	Timestamp    time.Time `json:"timestamp"`
}

// Process - информация о процессе
type Process struct {
	Name       string  `json:"name"`
	PID        int32   `json:"pid"`
	CPUPercent float64 `json:"cpu_percent"`
}
