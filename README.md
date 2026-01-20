# Sono APK Version Service

A Go service for managing APK uploads and version checking for the Sono app.

## Quick Start with Docker

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your settings (at minimum set WEBHOOK_SECRET)
nano .env

# Start all services
docker compose up -d
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| app | 8080 | Version service API |
| postgres | 5433 | PostgreSQL (for pgAdmin connection) |
| minio | 9000 | MinIO S3-compatible storage |

### Connecting to pgAdmin

Add a new server in pgAdmin:
- **Host**: `localhost`
- **Port**: `5433`
- **Database**: `sono_versions`
- **Username**: `sono` (or your POSTGRES_USER)
- **Password**: `sono` (or your POSTGRES_PASSWORD)

## Manual Setup (without Docker)

1. Install Go (1.21+)
2. Copy `.env.example` to `.env` and configure
3. Run the service:

```bash
go mod tidy
go run main.go
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/version/{channel}` | Get latest version for channel (stable/beta/nightly) |
| GET | `/api/v1/download/{channel}` | Download latest APK |
| GET | `/api/v1/stats` | Download statistics (requires database) |
| POST | `/api/v1/upload` | Upload new APK (requires webhook secret) |

## Upload Webhook

GitHub Actions can upload APKs via POST request:

```bash
curl -X POST https://your-domain.com/api/v1/upload \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: your-secret" \
  -d '{
    "channel": "beta",
    "version": "1.0.9",
    "version_code": 10,
    "release_notes": "Bug fixes",
    "apk_url": "https://github.com/.../sono-beta.apk"
  }'
```

## GitHub Actions Example

```yaml
- name: Upload APK to Version Service
  run: |
    curl -X POST ${{ secrets.VERSION_API_URL }}/api/v1/upload \
      -H "Content-Type: application/json" \
      -H "X-Webhook-Secret: ${{ secrets.WEBHOOK_SECRET }}" \
      -d '{
        "channel": "${{ matrix.flavor }}",
        "version": "${{ env.VERSION }}",
        "version_code": ${{ env.VERSION_CODE }},
        "release_notes": "${{ github.event.head_commit.message }}",
        "apk_url": "https://github.com/${{ github.repository }}/releases/download/v${{ env.VERSION }}/sono-${{ matrix.flavor }}.apk"
      }'
```

## Storage

The service supports:
- **Local**: Stores APKs in `./data/apks/`
- **S3**: S3-compatible storage (AWS S3, Cloudflare R2, MinIO)
- **Both**: S3 primary with local fallback

Configure via `STORAGE_TYPE` in `.env`.

## Database

PostgreSQL is used for tracking and logging:
- Download counts per version
- Upload history
- API request logs

Tables are auto-created via `init.sql` when using Docker.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Server port |
| BASE_URL | http://localhost:8080 | Public URL for download links |
| STORAGE_TYPE | both | Storage backend (s3/local/both) |
| WEBHOOK_SECRET | | Required for upload endpoint |
| DATABASE_URL | | PostgreSQL connection string |
| S3_ENDPOINT | | MinIO/S3 endpoint |
| S3_BUCKET | sono-apks | Bucket name |