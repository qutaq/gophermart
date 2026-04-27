CREATE TABLE IF NOT EXISTS orders (
    number TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'NEW',
    accrual BIGINT,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
