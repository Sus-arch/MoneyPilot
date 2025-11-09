# MoneyPilot

Проект в рамках хакатона VTB API 2025. Направление - Мультибанк: единый интерфейс финансового сервиса.

---

### Описание проекта

MoneyPilot — мультибанковское приложение, которое объединяет счета пользователя и помогает управлять ликвидностью.  
Сервис анализирует балансы и ставки через открытые банковские API, предлагая оптимальные действия: с какого счёта выгоднее платить и куда перевести свободные средства.  
Монетизация основана на премиум-подписке и партнёрских программах банков.

---

### Функционал

- Подключение нескольких банковских счетов через API VTB и других банков.
- Управление согласиями на просмотр, открытие и закрытие продуктов.
- Отслеживание балансов и ставок по счетам.
- Рекомендации по оптимизации расходов и переводам средств между счетами.
- Поддержка депозитов и карт (расширяемо на другие продукты).
- Реальное время подтверждений через WebSocket для согласий и подключений.

---

### Технический стек

**Backend:**  
- Go (Gin, Fiber)  
- Python (FastAPI, Pandas, Numpy)  

**Frontend:**  
- React, TypeScript, Vite.js  
- Chart.js, TailwindCSS  

**Database:**  
- PostgreSQL, Redis  

**DevOps:**  
- Docker, Docker Compose  

---

### Установка и запуск

1. Клонировать репозиторий:  
```bash
git clone https://github.com/yourusername/MoneyPilot.git
```
```
cd MoneyPilot
```
2. Перейти в папку deployments:
```
cd deployments/
```
3. Запустить проект через Docker Compose:
```
docker compose up --build
```
4. Добавть тестовые данные для одного пользователя:
```
# Добавляем VBank
docker compose exec -T postgres psql -U postgres -d finbalance -c "
WITH upsert_bank AS (
  INSERT INTO banks (code, name, api_base_url)
  VALUES ('vbank', 'Virtual Bank', 'https://vbank.open.bankingapi.ru')
  ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        api_base_url = EXCLUDED.api_base_url
  RETURNING id
),
bank AS (
  SELECT id FROM upsert_bank
  UNION ALL
  SELECT id FROM banks WHERE code='vbank' LIMIT 1
)
INSERT INTO users (client_id, bank_id, password_hash)
SELECT 'team081-1', bank.id, 'ddslFory8voO3gxZ2CEaQnHzLfv4HVzo' FROM bank
WHERE NOT EXISTS (
  SELECT 1 FROM users u WHERE u.client_id='team081-1' AND u.bank_id = bank.id
);"

# Добавляем SBank
docker compose exec -T postgres psql -U postgres -d finbalance -c "
WITH upsert_bank AS (
  INSERT INTO banks (code, name, api_base_url)
  VALUES ('sbank', 'Smart Bank', 'https://sbank.open.bankingapi.ru')
  ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        api_base_url = EXCLUDED.api_base_url
  RETURNING id
),
bank AS (
  SELECT id FROM upsert_bank
  UNION ALL
  SELECT id FROM banks WHERE code='sbank' LIMIT 1
)
INSERT INTO users (client_id, bank_id, password_hash)
SELECT 'team081-1', bank.id, 'ddslFory8voO3gxZ2CEaQnHzLfv4HVzo' FROM bank
WHERE NOT EXISTS (
  SELECT 1 FROM users u WHERE u.client_id='team081-1' AND u.bank_id = bank.id
);"

# Добавляем ABank
docker compose exec -T postgres psql -U postgres -d finbalance -c "
WITH upsert_bank AS (
  INSERT INTO banks (code, name, api_base_url)
  VALUES ('abank', 'Awesome Bank', 'https://abank.open.bankingapi.ru')
  ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        api_base_url = EXCLUDED.api_base_url
  RETURNING id
),
bank AS (
  SELECT id FROM upsert_bank
  UNION ALL
  SELECT id FROM banks WHERE code='abank' LIMIT 1
)
INSERT INTO users (client_id, bank_id, password_hash)
SELECT 'team081-1', bank.id, 'ddslFory8voO3gxZ2CEaQnHzLfv4HVzo' FROM bank
WHERE NOT EXISTS (
  SELECT 1 FROM users u WHERE u.client_id='team081-1' AND u.bank_id = bank.id
);"
```
5. Отркыть сайт по ссылке:
```
http://localhost:5173
```

### Команда

- **Андрей Харин** ([GitHub](https://github.com/Sus-arch)) – Frontend developer  

- **Владислав Ямщиков** ([GitHub](https://github.com/Vladik1605)) – Backend developer (GO)  

- **Алексей Кудрявцев** ([GitHub](https://github.com/Lookich39)) – Backend developer (Python) 

- **Полина Минченко** ([GitHub](https://github.com/Appolia9)) – Backend developer (Python)
  
- **Михаил Фомин** ([GitHub](https://github.com/Mikethrtn)) – Backend developer (Python)

