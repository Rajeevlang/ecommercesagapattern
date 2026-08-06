CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Trigger for updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Notifications Audit & Delivery Log Table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255) UNIQUE NOT NULL, -- Prevents duplicate notifications for same event
    user_id VARCHAR(255) NOT NULL,
    recipient VARCHAR(255) NOT NULL,              -- Email address or Phone number
    channel VARCHAR(50) NOT NULL,                 -- 'EMAIL', 'SMS', 'PUSH'
    template_name VARCHAR(100) NOT NULL,          -- e.g. 'order_confirmation'
    subject VARCHAR(255) DEFAULT '',
    content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',-- 'PENDING', 'SENT', 'FAILED', 'RETRYING'
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    error_message TEXT DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS set_notifications_updated_at ON notifications;
CREATE TRIGGER set_notifications_updated_at
BEFORE UPDATE ON notifications
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Indexes for fast querying & background worker queue polling
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status) WHERE status IN ('PENDING', 'RETRYING');
CREATE INDEX IF NOT EXISTS idx_notifications_idempotency ON notifications(idempotency_key);
