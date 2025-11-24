let isOnline = true;
let intervalId = null;
const statusIndicator = document.getElementById('status-indicator');
const statusText = document.getElementById('status-text');

function updateStatus(online) {
    if (isOnline === online) return;
    
    isOnline = online;
    
    if (online) {
        statusIndicator.className = 'status-indicator';
        statusIndicator.style.backgroundColor = '#4CAF50';
        statusText.textContent = 'Сервер доступен';
        statusText.style.color = '#4CAF50';
    } else {
        statusIndicator.className = 'status-indicator status-error';
        statusIndicator.style.backgroundColor = '#f44336';
        statusText.textContent = 'Сервер недоступен';
        statusText.style.color = '#c62828';
    }
}

async function fetchData() {
    try {
        const [latestRes, metricsRes] = await Promise.all([
            fetch('/api/latest', { timeout: 5000 }),
            fetch('/api/metrics', { timeout: 5000 })
        ]);
        
        if (!latestRes.ok || !metricsRes.ok) {
            throw new Error('Сервер вернул ошибку');
        }
        
        const latest = await latestRes.json();
        const metrics = await metricsRes.json();
        
        updateStatus(true);
        
        // Обновляем CPU
        if (latest.cpu_load !== undefined) {
            const cpuPercent = Math.min(100, Math.max(0, latest.cpu_load));
            document.getElementById('cpu-progress').style.width = cpuPercent + '%';
            document.getElementById('cpu-progress').textContent = cpuPercent.toFixed(1) + '%';
            document.getElementById('cpu-value').textContent = cpuPercent.toFixed(1) + '%';
        }
        
        // Обновляем GPU
        if (latest.gpu_load !== undefined) {
            const gpuPercent = Math.min(100, Math.max(0, latest.gpu_load));
            document.getElementById('gpu-progress').style.width = gpuPercent + '%';
            document.getElementById('gpu-progress').textContent = gpuPercent.toFixed(1) + '%';
            document.getElementById('gpu-value').textContent = gpuPercent.toFixed(1) + '%';
        }
        
        // Обновляем время
        if (latest.timestamp) {
            try {
                const timestamp = new Date(latest.timestamp);
                if (!isNaN(timestamp.getTime())) {
                    document.getElementById('timestamp').textContent = timestamp.toLocaleTimeString('ru-RU', {
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit'
                    });
                }
            } catch (e) {
                console.warn('Ошибка парсинга времени:', e);
            }
        }
        
        // Обновляем процессы
        const tableBody = document.getElementById('processes-table');
        tableBody.innerHTML = '';
        
        if (latest.top_processes && Array.isArray(latest.top_processes) && latest.top_processes.length > 0) {
            latest.top_processes.slice(0, 10).forEach((process, index) => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${process.name ? process.name.replace(/[<>]/g, '') : 'Неизвестный процесс'}</td>
                    <td>${process.pid || '-'}</td>
                    <td>${(process.cpu_percent || 0).toFixed(1)}%</td>
                `;
                tableBody.appendChild(row);
                
                // Добавляем анимацию появления
                row.style.opacity = '0';
                row.style.transform = 'translateY(10px)';
                setTimeout(() => {
                    row.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
                    row.style.opacity = '1';
                    row.style.transform = 'translateY(0)';
                }, 50 * index);
            });
        } else {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="3" class="no-data">Нет активных процессов с высокой загрузкой</td>';
            tableBody.appendChild(row);
        }
        
        // Обновляем средние значения
        if (metrics && typeof metrics.avg_cpu_load === 'number') {
            document.getElementById('avg-cpu').textContent = Math.min(100, Math.max(0, metrics.avg_cpu_load)).toFixed(1);
        }
        
        if (metrics && typeof metrics.avg_gpu_load === 'number') {
            document.getElementById('avg-gpu').textContent = Math.min(100, Math.max(0, metrics.avg_gpu_load)).toFixed(1);
        }
        
    } catch (error) {
        console.error('Ошибка при загрузке данных:', error);
        updateStatus(false);
        
        // Показываем ошибку в таблице только если это первая ошибка
        const tableBody = document.getElementById('processes-table');
        if (tableBody.innerHTML.includes('loading') || tableBody.innerHTML === '') {
            tableBody.innerHTML = `
                <tr>
                    <td colspan="3" class="error-message">
                        <strong>❌ Ошибка подключения к серверу</strong><br>
                        ${error.message || 'Не удалось загрузить данные'}
                    </td>
                </tr>
            `;
        }
    }
}

// Начальная загрузка
fetchData();

// Регулярное обновление
intervalId = setInterval(fetchData, 5000);

// Остановка интервала при уходе со страницы
window.addEventListener('beforeunload', () => {
    if (intervalId) {
        clearInterval(intervalId);
    }
});

// Обработка видимости вкладки
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        if (intervalId) {
            clearInterval(intervalId);
            intervalId = null;
        }
    } else {
        fetchData();
        if (!intervalId) {
            intervalId = setInterval(fetchData, 5000);
        }
    }
});

// Обработка ошибок сети
window.addEventListener('offline', () => {
    updateStatus(false);
    document.getElementById('processes-table').innerHTML = `
        <tr>
            <td colspan="3" class="error-message">
                <strong>❌ Нет интернет-соединения</strong><br>
                Проверьте подключение к сети
            </td>
        </tr>
    `;
});

window.addEventListener('online', () => {
    updateStatus(true);
    fetchData();
});