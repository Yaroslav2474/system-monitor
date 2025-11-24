package collector

import (
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo - временная структура для сортировки
type ProcessInfo struct {
	Name string
	PID  int32
	CPU  float64
}

// GetTopProcesses возвращает топ N процессов по загрузке CPU
func GetTopProcesses(n int) ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var results []ProcessInfo

	// Собираем информацию о процессах
	for _, p := range procs {
		// Пропускаем системные процессы без имени
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		// Получаем загрузку CPU
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			continue
		}

		// Пропускаем процессы с нулевой загрузкой
		if cpuPercent < 0.1 {
			continue
		}

		results = append(results, ProcessInfo{
			Name: name,
			PID:  p.Pid,
			CPU:  cpuPercent,
		})

		// Небольшая задержка, чтобы не нагружать систему
		time.Sleep(10 * time.Millisecond)
	}

	// Сортируем по загрузке CPU в убывающем порядке
	sort.Slice(results, func(i, j int) bool {
		return results[i].CPU > results[j].CPU
	})

	// Ограничиваем количеством
	if len(results) > n {
		return results[:n], nil
	}

	return results, nil
}
