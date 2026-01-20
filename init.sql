-- Version releases table
CREATE TABLE IF NOT EXISTS releases (
    id SERIAL PRIMARY KEY,
    channel VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    version_code INTEGER NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT,
    sha256 VARCHAR(64),
    release_notes TEXT,
    published_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(channel, version)
);

-- Download tracking table
CREATE TABLE IF NOT EXISTS downloads (
    id SERIAL PRIMARY KEY,
    release_id INTEGER REFERENCES releases(id),
    channel VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    downloaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Upload logs table
CREATE TABLE IF NOT EXISTS upload_logs (
    id SERIAL PRIMARY KEY,
    channel VARCHAR(20) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT,
    source_url TEXT,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API request logs table
CREATE TABLE IF NOT EXISTS request_logs (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER,
    ip_address INET,
    user_agent TEXT,
    response_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_releases_channel ON releases(channel);
CREATE INDEX IF NOT EXISTS idx_downloads_channel ON downloads(channel);
CREATE INDEX IF NOT EXISTS idx_downloads_date ON downloads(downloaded_at);
CREATE INDEX IF NOT EXISTS idx_request_logs_endpoint ON request_logs(endpoint);
CREATE INDEX IF NOT EXISTS idx_request_logs_date ON request_logs(created_at);

-- View for download statistics
CREATE OR REPLACE VIEW download_stats AS
SELECT
    channel,
    version,
    COUNT(*) as download_count,
    MIN(downloaded_at) as first_download,
    MAX(downloaded_at) as last_download
FROM downloads
GROUP BY channel, version
ORDER BY last_download DESC;

-- View for daily download counts
CREATE OR REPLACE VIEW daily_downloads AS
SELECT
    DATE(downloaded_at) as date,
    channel,
    COUNT(*) as downloads
FROM downloads
GROUP BY DATE(downloaded_at), channel
ORDER BY date DESC;