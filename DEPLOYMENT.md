# Руководство по развертыванию MoneyPilot на сервере

Это руководство описывает, как настроить и запустить приложение MoneyPilot на сервере (не на localhost).

## Изменения в коде

Для поддержки развертывания на сервере были внесены следующие изменения:

1. **CORS origins** теперь конфигурируются через переменную окружения `CORS_ORIGINS`
2. **WebSocket URL** во фронтенде теперь конфигурируется через переменную окружения `VITE_WS_URL`
3. **API URL** во фронтенде уже был конфигурируемым через `VITE_API_URL`
4. Все настройки подключения к БД и сервисам используют переменные окружения

## Конфигурация через переменные окружения

Создайте файл `.env` в корне проекта со следующими переменными:

### Обязательные переменные для Production

```env
# ============================================
# Go API Backend
# ============================================
SERVER_PORT=8080

# DSN для подключения к PostgreSQL
# Для Docker Compose используйте: postgres://postgres:balance@postgres:5432/finbalance?sslmode=disable
# Для внешней БД: postgres://user:password@host:port/database?sslmode=disable
POSTGRES_DSN=postgres://postgres:balance@postgres:5432/finbalance?sslmode=disable

# Адрес Redis сервера
# Для Docker Compose используйте: redis:6379
# Для внешнего Redis: host:port
REDIS_ADDR=redis:6379

# URL ML Engine сервиса
# Для Docker Compose используйте: http://mlengine:8000
# Для внешнего сервиса: http://your-ml-service-host:8000
ML_URL=http://mlengine:8000

# Учетные данные для банковского API
LOGIN_HAC=team081
PASSWORD_HAC=your-password-here

# ============================================
# CORS Configuration
# ============================================
# Разрешенные origins для CORS (разделенные запятой)
# ВАЖНО: Для production замените на ваш реальный домен
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# ============================================
# Frontend (React)
# ============================================
# URL Go API сервера (используется фронтендом)
# Для production: https://api.yourdomain.com или https://yourdomain.com/api
VITE_API_URL=https://api.yourdomain.com

# WebSocket URL для фронтенда
# Для production с SSL: wss://api.yourdomain.com/ws
# Если не указан, будет автоматически выведен из VITE_API_URL
VITE_WS_URL=wss://api.yourdomain.com/ws
```

## Варианты развертывания

### 1. Локальная разработка (вне Docker)

Используйте следующие настройки в `.env`:

```env
POSTGRES_DSN=postgres://postgres:balance@localhost:5432/finbalance?sslmode=disable
REDIS_ADDR=localhost:6379
ML_URL=http://localhost:8000
CORS_ORIGINS=http://localhost:5173
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
```

### 2. Docker Compose (рекомендуется)

При использовании Docker Compose, сервисы внутри Docker сети обращаются друг к другу по именам сервисов:

```env
POSTGRES_DSN=postgres://postgres:balance@postgres:5432/finbalance?sslmode=disable
REDIS_ADDR=redis:6379
ML_URL=http://mlengine:8000
CORS_ORIGINS=http://localhost:5173,https://yourdomain.com
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
```

**Важно**: `VITE_API_URL` для фронтенда должен быть доступен из браузера пользователя, поэтому используйте публичный URL (localhost:8080 если обращаетесь с той же машины, или домен если с другого устройства).

### 3. Production на сервере с доменом

#### Вариант A: Все сервисы на одном домене с Reverse Proxy (Nginx)

Настройте Nginx для проксирования запросов:

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    # Frontend
    location / {
        proxy_pass http://localhost:5173;
    }

    # API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

`.env` для этого случая:

```env
POSTGRES_DSN=postgres://user:password@localhost:5432/finbalance?sslmode=require
REDIS_ADDR=localhost:6379
ML_URL=http://localhost:8000
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
VITE_API_URL=https://yourdomain.com
VITE_WS_URL=wss://yourdomain.com/ws
```

#### Вариант B: Раздельные поддомены

```env
POSTGRES_DSN=postgres://user:password@db.yourdomain.com:5432/finbalance?sslmode=require
REDIS_ADDR=redis.yourdomain.com:6379
ML_URL=http://ml.yourdomain.com:8000
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
VITE_API_URL=https://api.yourdomain.com
VITE_WS_URL=wss://api.yourdomain.com/ws
```

## Запуск приложения

### С Docker Compose

1. Создайте файл `.env` в корне проекта с нужными значениями
2. Перейдите в папку deployments:
   ```bash
   cd deployments/
   ```
3. Запустите сервисы:
   ```bash
   docker compose up --build
   ```

### Без Docker

1. Убедитесь, что PostgreSQL и Redis запущены
2. Запустите ML Engine:
   ```bash
   cd ml_engine
   python -m uvicorn app:app --host 0.0.0.0 --port 8000
   ```
3. Запустите Go API:
   ```bash
   go run cmd/api/main.go
   ```
4. Запустите фронтенд:
   ```bash
   cd web
   npm install
   npm run dev
   ```

## Проверка конфигурации

После запуска проверьте:

1. **API доступен**: `curl http://your-api-url:8080/api/auth/login`
2. **Frontend доступен**: Откройте в браузере `http://your-frontend-url:5173`
3. **CORS работает**: Проверьте в консоли браузера, нет ли ошибок CORS
4. **WebSocket подключен**: Проверьте в Network tab браузера, что WebSocket подключается успешно

## Безопасность

⚠️ **Важные моменты для production**:

1. Используйте HTTPS для всех публичных сервисов
2. Настройте SSL сертификаты (Let's Encrypt)
3. Не храните пароли в `.env` в открытом виде - используйте секреты Docker или менеджеры секретов
4. Ограничьте `CORS_ORIGINS` только вашими доменами
5. Используйте сильные пароли для БД
6. Настройте файрвол для ограничения доступа к портам БД и Redis

## Решение проблем

### Ошибка CORS

Убедитесь, что `CORS_ORIGINS` в `.env` содержит точный URL фронтенда (включая протокол и порт).

### WebSocket не подключается

Проверьте:
- `VITE_WS_URL` указан правильно
- Reverse proxy настроен для WebSocket (если используется)
- Файрвол не блокирует WebSocket соединения

### API недоступен из фронтенда

Проверьте:
- `VITE_API_URL` указан правильно
- API сервер запущен и доступен
- Нет проблем с сетью/файрволом

