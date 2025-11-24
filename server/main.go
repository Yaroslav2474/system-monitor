package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"system-monitor/client/models"
)

type Server struct {
	data    models.MonitorData
	mutex   sync.RWMutex
	storage []models.MonitorData
}

func NewServer() *Server {
	return &Server{
		storage: make([]models.MonitorData, 0, 100),
	}
}

func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var data models.MonitorData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Неверный JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	s.mutex.Lock()
	s.data = data
	s.storage = append(s.storage, data)
	// Ограничиваем хранилище последними 100 записями
	if len(s.storage) > 100 {
		s.storage = s.storage[len(s.storage)-100:]
	}
	s.mutex.Unlock()

	log.Printf("Получены данные: CPU=%.1f%%, GPU=%.1f%%, Процессов=%d",
		data.CPULoad, data.GPULoad, len(data.TopProcesses))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.data.Timestamp.IsZero() {
		http.Error(w, "Нет данных", http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.data)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.storage)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(s.storage) == 0 {
		http.Error(w, "Нет данных", http.StatusNoContent)
		return
	}

	// Вычисляем средние значения за последнюю минуту
	var (
		cpuSum, gpuSum float64
		count          int
		now            = time.Now()
	)

	for i := len(s.storage) - 1; i >= 0; i-- {
		if now.Sub(s.storage[i].Timestamp).Minutes() > 1 {
			break
		}
		cpuSum += s.storage[i].CPULoad
		gpuSum += s.storage[i].GPULoad
		count++
	}

	if count == 0 {
		http.Error(w, "Нет данных за последнюю минуту", http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{
		"avg_cpu_load": cpuSum / float64(count),
		"avg_gpu_load": gpuSum / float64(count),
	})
}

// getStaticDir определяет правильный путь к папке static
func getStaticDir() string {
	// Пытаемся найти папку static в разных возможных местах
	possiblePaths := []string{
		"./static",        // запуск из папки server
		"./server/static", // запуск из корня проекта
		"static",          // относительный путь
		"../static",       // если запускаем из другой папки
		"./build/static",  // для production сборки
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				log.Printf("✅ Найдена папка static: %s", absPath)
				return absPath
			}
		}
	}

	// Если ничего не найдено, используем текущую директорию
	currentDir, _ := os.Getwd()
	log.Printf("⚠️ Папка static не найдена, использую текущую директорию: %s", currentDir)
	return currentDir
}

func main() {
	server := NewServer()
	staticDir := getStaticDir()

	// Получаем текущую директорию для отладки
	currentDir, _ := os.Getwd()

	log.Printf("📁 Папка со статическими файлами: %s", staticDir)
	log.Printf("📍 Текущая рабочая директория: %s", currentDir)

	// API эндпоинты
	http.HandleFunc("/api/monitor", server.handleMonitor)
	http.HandleFunc("/api/latest", server.handleLatest)
	http.HandleFunc("/api/history", server.handleHistory)
	http.HandleFunc("/api/metrics", server.handleMetrics)

	// Обслуживание статических файлов
	// Правильный способ: ищем index.html в корне staticDir
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Главная страница
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Проверяем несколько возможных путей к index.html
		possibleIndexPaths := []string{
			filepath.Join(staticDir, "index.html"),
			filepath.Join(staticDir, "static", "index.html"),
			filepath.Join(currentDir, "static", "index.html"),
			filepath.Join(currentDir, "server", "static", "index.html"),
			filepath.Join(staticDir, "..", "static", "index.html"),
		}

		var indexPath string
		var fileExists bool

		for _, path := range possibleIndexPaths {
			if _, err := os.Stat(path); err == nil {
				indexPath = path
				fileExists = true
				log.Printf("✅ Найден index.html: %s", indexPath)
				break
			}
		}

		if !fileExists {
			// Показываем подробную отладочную страницу
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)

			// Формируем список проверенных путей
			var pathsHTML string
			for _, path := range possibleIndexPaths {
				pathsHTML += "<li><code>" + path + "</code></li>\n"
			}

			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Ошибка 404 - Файл не найден</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px; background: white; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #d32f2f; margin-bottom: 20px; }
        .error { background-color: #ffebee; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #f44336; }
        .paths { background-color: #e8f5e8; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #4caf50; }
        .debug { background-color: #e3f2fd; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #2196f3; }
        .solution { background-color: #fff8e1; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #ff9800; }
        code { background-color: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
        pre { background-color: #2d2d2d; color: #f8f8f2; padding: 15px; border-radius: 4px; overflow-x: auto; }
        ul { padding-left: 20px; margin: 10px 0; }
        li { margin: 5px 0; }
        .structure { font-family: monospace; white-space: pre; background-color: #f5f5f5; padding: 15px; border-radius: 4px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>❌ Файл index.html не найден</h1>
        <p>Сервер не может найти главную страницу веб-интерфейса.</p>
        
        <div class="error">
            <h3>🔍 Проверенные пути:</h3>
            <ul>
                ` + pathsHTML + `
            </ul>
        </div>
        
        <div class="debug">
            <h3>⚙️ Отладочная информация:</h3>
            <p><strong>Текущая рабочая директория:</strong> <code>` + currentDir + `</code></p>
            <p><strong>Определенная папка static:</strong> <code>` + staticDir + `</code></p>
            <p><strong>Рабочая директория Go:</strong> <code>` + filepath.Dir(os.Args[0]) + `</code></p>
        </div>
        
        <div class="solution">
            <h3>🛠️ Как исправить:</h3>
            <ol>
                <li><strong>Проверьте структуру проекта:</strong>
                    <div class="structure">
system-monitor/
├── server/
│   ├── main.go
│   └── static/
│       ├── index.html
│       ├── css/
│       │   └── styles.css
│       └── js/
│           └── script.js
                    </div>
                </li>
                <li><strong>Запустите сервер из правильной директории:</strong>
                    <pre>cd system-monitor/server
go run main.go</pre>
                </li>
                <li><strong>Или создайте недостающие файлы:</strong>
                    <pre>mkdir -p server/static/css server/static/js
# Скопируйте файлы index.html, styles.css, script.js в соответствующие папки</pre>
                </li>
                <li><strong>Проверьте права доступа:</strong>
                    <pre>ls -la server/static/</pre>
                </li>
            </ol>
        </div>
        
        <div class="paths">
            <h3>📦 Альтернативные варианты:</h3>
            <p>Если вы хотите использовать другую структуру папок, измените код в <code>main.go</code> в функции <code>main()</code>, в разделе поиска путей к <code>index.html</code>.</p>
        </div>
    </div>
</body>
</html>
			`))
			return
		}

		// Обслуживаем главную страницу
		log.Printf("✅ Отправка файла: %s", indexPath)
		http.ServeFile(w, r, indexPath)
	})

	// Эндпоинт здоровья
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"server":      "running",
			"time":        time.Now().Format(time.RFC3339),
			"static_dir":  staticDir,
			"current_dir": currentDir,
			"port":        "8080",
		})
	})

	// Эндпоинт для проверки статических файлов
	http.HandleFunc("/debug/static", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Проверяем доступные файлы
		files := []string{}
		err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, _ := filepath.Rel(staticDir, path)
				files = append(files, relPath)
			}
			return nil
		})

		var filesHTML string
		if err == nil {
			for _, file := range files {
				filesHTML += "<li><code>" + file + "</code></li>\n"
			}
		} else {
			filesHTML = "<li>Ошибка при сканировании папки: " + err.Error() + "</li>"
		}

		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Отладка статических файлов</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; padding: 20px; border: 1px solid #e0e0e0; border-radius: 8px; background: white; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2196f3; margin-bottom: 20px; }
        .info { background-color: #e3f2fd; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #2196f3; }
        .files { background-color: #e8f5e8; padding: 15px; border-radius: 4px; margin: 20px 0; border-left: 4px solid #4caf50; }
        code { background-color: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 Отладка статических файлов</h1>
        
        <div class="info">
            <h3>📁 Информация о директориях:</h3>
            <p><strong>Папка static:</strong> <code>` + staticDir + `</code></p>
            <p><strong>Текущая директория:</strong> <code>` + currentDir + `</code></p>
        </div>
        
        <div class="files">
            <h3>📋 Найденные файлы:</h3>
            <ul>
                ` + filesHTML + `
            </ul>
        </div>
        
        <div class="info">
            <h3>🔗 Полезные ссылки:</h3>
            <ul>
                <li><a href="/">Главная страница</a></li>
                <li><a href="/health">Эндпоинт здоровья</a></li>
                <li><a href="/static/index.html">Прямая ссылка на index.html</a></li>
            </ul>
        </div>
    </div>
</body>
</html>
		`))
	})

	log.Println("🚀 Сервер запущен на http://localhost:8080")
	log.Println("📊 Веб-интерфейс: http://localhost:8080")
	log.Println("🔧 Эндпоинт здоровья: http://localhost:8080/health")
	log.Println("🔍 Отладка файлов: http://localhost:8080/debug/static")
	log.Println("📡 API эндпоинты:")
	log.Println("   POST /api/monitor  - прием данных мониторинга")
	log.Println("   GET  /api/latest   - последние полученные данные")
	log.Println("   GET  /api/history  - история последних 100 записей")
	log.Println("   GET  /api/metrics  - средние показатели за последнюю минуту")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
