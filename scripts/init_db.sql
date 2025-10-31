-- Users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_premium BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Banks (справочник)
CREATE TABLE banks (
    id SERIAL PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,  -- vbank, smartbank, awesomebank
    name VARCHAR(128) NOT NULL,
    api_base_url TEXT NOT NULL
);

-- Accounts
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    account_number VARCHAR(64),
    currency VARCHAR(16),
    balance NUMERIC(18,2) DEFAULT 0,
    rate NUMERIC(5,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Transactions
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id INT REFERENCES accounts(id) ON DELETE CASCADE,
    amount NUMERIC(18,2) NOT NULL,
    currency VARCHAR(16),
    description TEXT,
    category VARCHAR(64),
    date TIMESTAMP DEFAULT NOW()
);

-- Consents (OAuth-разрешения пользователя)
CREATE TABLE consents (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    bank_id INT REFERENCES banks(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recommendations (от Python-сервиса)
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    benefit NUMERIC(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Leads (заявки на продукты)
CREATE TABLE leads (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    product_id VARCHAR(64),
    amount NUMERIC(18,2),
    status VARCHAR(32) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);
