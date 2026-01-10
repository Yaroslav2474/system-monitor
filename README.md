# System Monitor

Веб-сервер для мониторинга системы Windows 10 с поддержкой отслеживания ГПУ.

## 🛠️ Требования

- Windows 10
- Go 1.21+
- OpenHardwareMonitor (для мониторинга GPU)

## 📁 Структура проекта
system-monitor/
├── go.mod # Зависимости с проверенными версиями
├── go.sum
├── README.md
├── client/ # Клиент для сбора данных
│ ├── main.go # Основной клиент
│ ├── collector/ # Модули сбора данных
│ │ ├── cpu_windows.go
│ │ ├── gpu_windows.go
│ │ └── processes_windows.go
│ ├── models/ # Структуры данных
│ │ └── data.go
│ └── sender/ # Отправка данных
│ └── sender.go
└── server/ # Веб-сервер
└── main.go


## ⚙️ Настройка

### 1. Установка OpenHardwareMonitor

1. Скачайте [OpenHardwareMonitor](https://openhardwaremonitor.org/downloads/)
2. Запустите `OpenHardwareMonitor.exe`
3. В меню Options → включите "Start HTTP Server" (порт 8085)

### 2. Настройка Go модулей

```bash
go mod init system-monitor
go mod tidy


http://localhost:8080 

