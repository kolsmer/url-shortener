ALTER TABLE urls ADD COLUMN user_id INTEGER REFERENCES users(id);
CREATE INDEX IF NOT EXISTS idx_user_id ON urls(user_id);