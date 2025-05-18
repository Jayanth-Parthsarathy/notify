CREATE TABLE IF NOT EXISTS dlq_messages (
    id SERIAL PRIMARY KEY,
    message_id TEXT,
    body JSONB,
    headers JSONB,
    received_at TIMESTAMPTZ DEFAULT now()
);
