-- ========================
-- 💾 INIT_DB.SQL
-- ========================

-- Очистка при пересоздании
DROP TABLE IF EXISTS api_cache, product_agreements, products, payments, payment_consents, account_consents, transactions, accounts, users, banks, product_agreement_consents CASCADE;

-- 🏦 Таблица банков
CREATE TABLE banks (
    id SERIAL PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,  -- vbank, sbank, abank
    name VARCHAR(128) NOT NULL,
    api_base_url TEXT NOT NULL,
    jwks_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 👤 Пользователи
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    client_id VARCHAR(64) NOT NULL,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    email VARCHAR(255),
    password_hash TEXT NOT NULL,
    segment VARCHAR(32),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 💳 Счета
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    account_number VARCHAR(64) NOT NULL,
    account_type VARCHAR(32) DEFAULT 'checking',
    nickname VARCHAR(64),
    currency VARCHAR(8) DEFAULT 'RUB',
    balance NUMERIC(18,2) DEFAULT 0,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, bank_id, account_number)
);

-- 📈 Транзакции
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id INT REFERENCES accounts(id) ON DELETE CASCADE,
    amount NUMERIC(18,2) NOT NULL,
    currency VARCHAR(8),
    description TEXT,
    category VARCHAR(64),
    booking_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 🪪 Согласия (доступ к данным)
CREATE TABLE account_consents (
    id SERIAL PRIMARY KEY,
    consent_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    requesting_bank VARCHAR(64),
    permissions TEXT[],
    status VARCHAR(32) DEFAULT 'approved',
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 💸 Платежные согласия
CREATE TABLE payment_consents (
    id SERIAL PRIMARY KEY,
    consent_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INT REFERENCES users(id),
    bank_id INT REFERENCES banks(id),
    consent_type VARCHAR(32),
    amount NUMERIC(18,2),
    debtor_account VARCHAR(64),
    creditor_account VARCHAR(64),
    valid_until TIMESTAMP,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 💰 Платежи
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INT REFERENCES users(id),
    debtor_account VARCHAR(64),
    creditor_account VARCHAR(64),
    amount NUMERIC(18,2),
    currency VARCHAR(8),
    status VARCHAR(32) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 🧩 Продукты
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    product_id VARCHAR(64) UNIQUE NOT NULL,
    bank_id INT REFERENCES banks(id),
    product_type VARCHAR(32),
    name VARCHAR(128),
    description TEXT,
    interest_rate NUMERIC(5,2),
    min_amount NUMERIC(18,2),
    max_amount NUMERIC(18,2),
    term_months INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 📜 Договоры по продуктам
CREATE TABLE product_agreements (
    id SERIAL PRIMARY KEY,
    agreement_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INT REFERENCES users(id),
    product_id VARCHAR(64) REFERENCES products(product_id),
    amount NUMERIC(18,2),
    term_months INT,
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);



CREATE TABLE product_agreement_consents (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(64) UNIQUE NOT NULL,
    consent_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    requesting_bank VARCHAR(64),
    read_product_agreements BOOLEAN DEFAULT false,
    open_product_agreements BOOLEAN DEFAULT false,
    close_product_agreements BOOLEAN DEFAULT false,
    allowed_product_types TEXT[] DEFAULT '{}',
    max_amount NUMERIC(18,2),
    status VARCHAR(32) DEFAULT 'pending',
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);





-- 💾 Кэш для API ответов
CREATE TABLE api_cache (
    id SERIAL PRIMARY KEY,
    cache_key VARCHAR(255) UNIQUE NOT NULL,
    data JSONB NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_bank_id ON users(bank_id);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_api_cache_key ON api_cache(cache_key);
CREATE INDEX idx_api_cache_expires_at ON api_cache(expires_at);


-- Cоздание 10 пользователей

-- Добавляем VBank и 10 пользователей team081-1...team081-10
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
SELECT 'team081-' || n, bank.id, 'Nx1FIkTkSG2Sxk9R'
FROM bank, generate_series(1, 10) AS gs(n)
WHERE NOT EXISTS (
  SELECT 1 FROM users u
  WHERE u.client_id = 'team081-' || n AND u.bank_id = bank.id
);

-- Добавляем SBank и 10 пользователей team081-1...team081-10
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
SELECT 'team081-' || n, bank.id, 'Nx1FIkTkSG2Sxk9R'
FROM bank, generate_series(1, 10) AS gs(n)
WHERE NOT EXISTS (
  SELECT 1 FROM users u
  WHERE u.client_id = 'team081-' || n AND u.bank_id = bank.id
);

-- Добавляем ABank и 10 пользователей team081-1...team081-10
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
SELECT 'team081-' || n, bank.id, 'Nx1FIkTkSG2Sxk9R'
FROM bank, generate_series(1, 10) AS gs(n)
WHERE NOT EXISTS (
  SELECT 1 FROM users u
  WHERE u.client_id = 'team081-' || n AND u.bank_id = bank.id
);
