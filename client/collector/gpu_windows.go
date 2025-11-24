package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GetGPULoad возвращает загрузку GPU для AMD RX 580
// Использует OpenHardwareMonitor HTTP API
func GetGPULoad() (float64, error) {
	// Сначала пробуем через OpenHardwareMonitor
	if gpuLoad, err := getGPUFromOHM(); err == nil && gpuLoad >= 0 {
		return gpuLoad, nil
	}

	// Если OHM недоступен, пробуем через PowerShell
	return getGPUFromPowerShell()
}

// getGPUFromOHM получает данные из OpenHardwareMonitor
func getGPUFromOHM() (float64, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8085/data.json")
	if err != nil {
		return -1, fmt.Errorf("OHM недоступен: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("OHM вернул статус %d", resp.StatusCode)
	}

	// Упрощенный парсинг JSON для GPU Load
	var sensors []struct {
		Text  string      `json:"Text"`
		Value interface{} `json:"Value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sensors); err != nil {
		return -1, fmt.Errorf("ошибка парсинга OHM: %v", err)
	}

	// Ищем датчик GPU Load
	for _, sensor := range sensors {
		if strings.Contains(sensor.Text, "GPU Core") && strings.Contains(sensor.Text, "Load") {
			switch v := sensor.Value.(type) {
			case float64:
				return v, nil
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					return f, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("датчик GPU Load не найден")
}

// getGPUFromPowerShell получает загрузку GPU через PowerShell
func getGPUFromPowerShell() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-Command",
		"Get-Counter '\\GPU Engine(*)\\Utilization Percentage' | Select-Object -ExpandProperty CounterSamples | Where-Object { $_.InstanceName -like '*3D*' } | Select-Object -First 1 -ExpandProperty CookedValue")

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ошибка PowerShell: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return 0, fmt.Errorf("пустой ответ от PowerShell")
	}

	value, err := strconv.ParseFloat(outputStr, 64)
	if err != nil {
		return 0, fmt.Errorf("ошибка парсинга значения: %v", err)
	}

	return value, nil
}
