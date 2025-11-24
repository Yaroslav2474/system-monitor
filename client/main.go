package main

import (
	"log"
	"os"
	"time"

	"system-monitor/client/collector"
	"system-monitor/client/models"
	"system-monitor/client/sender"
)

const (
	ServerURL = "http://localhost:8080"
	Interval  = 5 * time.Second
)

func main() {
	log.Println("🚀 Запуск клиента мониторинга системы...")
	log.Println("📈 Сбор данных каждые", Interval)

	for {
		data, err := collectSystemData()
		if err != nil {
			log.Printf("❌ Ошибка сбора данных: %v", err)
			time.Sleep(Interval)
			continue
		}

		if err := sender.SendToServer(data, ServerURL); err != nil {
			log.Printf("❌ Ошибка отправки данных: %v", err)
		}

		time.Sleep(Interval)
	}
}

func collectSystemData() (models.MonitorData, error) {
	var data models.MonitorData
	data.Timestamp = time.Now()

	// Сбор загрузки CPU
	cpuLoad, err := collector.GetCPULoad()
	if err != nil {
		log.Printf("⚠️ Ошибка сбора CPU: %v", err)
		cpuLoad = 0
	}
	data.CPULoad = cpuLoad

	// Сбор загрузки GPU
	gpuLoad, err := collector.GetGPULoad()
	if err != nil {
		log.Printf("⚠️ Ошибка сбора GPU: %v", err)
		gpuLoad = 0
	}
	data.GPULoad = gpuLoad

	// Сбор топ процессов
	processes, err := collector.GetTopProcesses(10)
	if err != nil {
		log.Printf("⚠️ Ошибка сбора процессов: %v", err)
	} else {
		for _, p := range processes {
			data.TopProcesses = append(data.TopProcesses, models.Process{
				Name:       p.Name,
				PID:        p.PID,
				CPUPercent: p.CPU,
			})
		}
	}

	log.Printf("📊 Собраны данные: CPU=%.1f%%, GPU=%.1f%%, Процессов=%d",
		cpuLoad, gpuLoad, len(data.TopProcesses))

	return data, nil
}

func init() {
	// Настройка логирования
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
}
