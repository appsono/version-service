package database

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func New(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		log.Println("Database: No DATABASE_URL configured, logging disabled")
		return nil, nil
	}

	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}

	log.Println("Database: Connected successfully")

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		log.Printf("Database: Warning - failed to initialize schema: %v", err)
	}

	return db, nil
}

func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema := `
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

	CREATE TABLE IF NOT EXISTS downloads (
		id SERIAL PRIMARY KEY,
		release_id INTEGER REFERENCES releases(id),
		channel VARCHAR(20) NOT NULL,
		version VARCHAR(50) NOT NULL,
		ip_address INET,
		user_agent TEXT,
		downloaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS upload_logs (
		id SERIAL PRIMARY KEY,
		channel VARCHAR(20) NOT NULL,
		version VARCHAR(50) NOT NULL,
		status VARCHAR(20) NOT NULL,
		message TEXT,
		source_url TEXT,
		uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

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

	CREATE INDEX IF NOT EXISTS idx_releases_channel ON releases(channel);
	CREATE INDEX IF NOT EXISTS idx_downloads_channel ON downloads(channel);
	CREATE INDEX IF NOT EXISTS idx_downloads_date ON downloads(downloaded_at);
	CREATE INDEX IF NOT EXISTS idx_request_logs_endpoint ON request_logs(endpoint);
	CREATE INDEX IF NOT EXISTS idx_request_logs_date ON request_logs(created_at);
	`

	_, err := db.conn.ExecContext(ctx, schema)
	if err != nil {
		return err
	}

	log.Println("Database: Schema initialized successfully")
	return nil
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

type Release struct {
	ID           int
	Channel      string
	Version      string
	VersionCode  int
	FileName     string
	FileSize     int64
	SHA256       string
	ReleaseNotes string
	PublishedAt  time.Time
}

func (db *DB) InsertRelease(ctx context.Context, r *Release) (int, error) {
	if db == nil || db.conn == nil {
		return 0, nil
	}

	var id int
	err := db.conn.QueryRowContext(ctx, `
		INSERT INTO releases (channel, version, version_code, file_name, file_size, sha256, release_notes, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (channel, version) DO UPDATE SET
			version_code = EXCLUDED.version_code,
			file_name = EXCLUDED.file_name,
			file_size = EXCLUDED.file_size,
			sha256 = EXCLUDED.sha256,
			release_notes = EXCLUDED.release_notes,
			published_at = EXCLUDED.published_at
		RETURNING id
	`, r.Channel, r.Version, r.VersionCode, r.FileName, r.FileSize, r.SHA256, r.ReleaseNotes, r.PublishedAt).Scan(&id)

	return id, err
}

func (db *DB) GetLatestRelease(ctx context.Context, channel string) (*Release, error) {
	if db == nil || db.conn == nil {
		return nil, nil
	}

	r := &Release{}
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, channel, version, version_code, file_name, file_size, sha256, release_notes, published_at
		FROM releases
		WHERE channel = $1
		ORDER BY published_at DESC
		LIMIT 1
	`, channel).Scan(&r.ID, &r.Channel, &r.Version, &r.VersionCode, &r.FileName, &r.FileSize, &r.SHA256, &r.ReleaseNotes, &r.PublishedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (db *DB) LogDownload(ctx context.Context, channel, version, ipAddress, userAgent string) error {
	if db == nil || db.conn == nil {
		return nil
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO downloads (channel, version, ip_address, user_agent, release_id)
		SELECT $1, $2, $3::inet, $4, id FROM releases WHERE channel = $1 AND version = $2
	`, channel, version, ipAddress, userAgent)

	return err
}

func (db *DB) LogUpload(ctx context.Context, channel, version, status, message, sourceURL string) error {
	if db == nil || db.conn == nil {
		return nil
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO upload_logs (channel, version, status, message, source_url)
		VALUES ($1, $2, $3, $4, $5)
	`, channel, version, status, message, sourceURL)

	return err
}

func (db *DB) LogRequest(ctx context.Context, endpoint, method string, statusCode int, ipAddress, userAgent string, responseTimeMs int) error {
	if db == nil || db.conn == nil {
		return nil
	}

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO request_logs (endpoint, method, status_code, ip_address, user_agent, response_time_ms)
		VALUES ($1, $2, $3, $4::inet, $5, $6)
	`, endpoint, method, statusCode, ipAddress, userAgent, responseTimeMs)

	return err
}

type DownloadStats struct {
	Channel       string
	Version       string
	DownloadCount int
	FirstDownload time.Time
	LastDownload  time.Time
}

func (db *DB) GetDownloadStats(ctx context.Context) ([]DownloadStats, error) {
	if db == nil || db.conn == nil {
		return nil, nil
	}

	rows, err := db.conn.QueryContext(ctx, `SELECT channel, version, download_count, first_download, last_download FROM download_stats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DownloadStats
	for rows.Next() {
		var s DownloadStats
		if err := rows.Scan(&s.Channel, &s.Version, &s.DownloadCount, &s.FirstDownload, &s.LastDownload); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}