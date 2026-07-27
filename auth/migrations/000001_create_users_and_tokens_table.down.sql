-- Rollback Auth Service Database Schema
DROP TABLE IF EXISTS login_activity;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

DROP TRIGGER IF EXISTS set_refresh_tokens_updated_at ON refresh_tokens;
DROP TRIGGER IF EXISTS set_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
