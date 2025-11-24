package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// GetCPULoad возвращает текущую загрузку CPU в процентах
func GetCPULoad() (float64, error) {
	// Используем интервал 500ms для точного измерения
	percent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return 0, err
	}

	if len(percent) > 0 {
		return percent[0], nil
	}

	return 0, nil
}
