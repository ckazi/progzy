-- Create tables for proxy server management

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    comment TEXT,
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    proxy_type VARCHAR(20) DEFAULT 'default',
    twofa_secret TEXT,
    twofa_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Proxy settings table
CREATE TABLE IF NOT EXISTS proxy_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Traffic statistics table
CREATE TABLE IF NOT EXISTS traffic_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    request_count INTEGER DEFAULT 0,
    date DATE DEFAULT CURRENT_DATE,
    UNIQUE(user_id, date)
);

-- Request logs table
CREATE TABLE IF NOT EXISTS request_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    method VARCHAR(10),
    url TEXT,
    status_code INTEGER,
    bytes_sent BIGINT,
    bytes_received BIGINT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_id ON traffic_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_date ON traffic_stats(date);

-- Insert default proxy settings
INSERT INTO proxy_settings (key, value, description) VALUES
    ('proxy_port', '8080', 'Port for proxy server'),
    ('max_connections', '1000', 'Maximum concurrent connections'),
    ('timeout_seconds', '30', 'Connection timeout in seconds'),
    ('enable_logging', 'true', 'Enable request logging'),
    ('allow_http', 'true', 'Allow HTTP connections'),
    ('allow_https', 'true', 'Allow HTTPS connections')
ON CONFLICT (key) DO NOTHING;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_proxy_settings_updated_at BEFORE UPDATE ON proxy_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Proxy filter tables
CREATE TABLE IF NOT EXISTS user_proxy_whitelist (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, value)
);

CREATE TABLE IF NOT EXISTS user_proxy_blacklist (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, value)
);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_twofa_backup_codes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    used_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS twofa_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    ip_address TEXT,
    event TEXT,
    method TEXT,
    success BOOLEAN DEFAULT FALSE,
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_twofa_logs_user ON twofa_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_twofa_logs_ip ON twofa_logs(ip_address, created_at);
