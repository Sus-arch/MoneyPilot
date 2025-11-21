# Развертывание MoneyPilot на сервере

## Краткое руководство

Для развертывания приложения на сервере (не localhost) были внесены следующие изменения:

### Что было изменено в коде:

1. ✅ **CORS настройки** - теперь конфигурируются через переменную окружения `CORS_ORIGINS`
2. ✅ **WebSocket URL** - теперь конфигурируется через переменную окружения `VITE_WS_URL`
3. ✅ **API URL** - уже был конфигурируемым через `VITE_API_URL`

### Быстрый старт

1. **Создайте файл `.env` в корне проекта** с следующими переменными:

```env
# Для Docker Compose (сервисы внутри Docker сети)
POSTGRES_DSN=postgres://postgres:balance@postgres:5432/finbalance?sslmode=disable
REDIS_ADDR=redis:6379
ML_URL=http://mlengine:8000

# ВАЖНО: Замените на ваш реальный домен для production
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# URL для фронтенда (должен быть доступен из браузера)
VITE_API_URL=https://api.yourdomain.com
VITE_WS_URL=wss://api.yourdomain.com/ws
```

2. **Для локальной разработки (вне Docker):**
```env
POSTGRES_DSN=postgres://postgres:balance@localhost:5432/finbalance?sslmode=disable
REDIS_ADDR=localhost:6379
ML_URL=http://localhost:8000
CORS_ORIGINS=http://localhost:5173
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
```

3. **Запустите приложение:**
```bash
cd deployments/
docker compose up --build
```

### Важные моменты

⚠️ **Для production обязательно:**
- Используйте HTTPS (wss:// для WebSocket, https:// для API)
- Настройте `CORS_ORIGINS` на ваш реальный домен
- Используйте сильные пароли для БД
- Не храните `.env` файл в репозитории (добавьте в `.gitignore`)

### Подробная документация

Полная документация доступна в файле [DEPLOYMENT.md](./DEPLOYMENT.md)

