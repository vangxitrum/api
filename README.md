## Stream Platform (VOD + Livestream)

A backend platform that provides **streaming services for a variety of content**, including:

- **VOD**: media upload/processing and playlist management
- **Livestream**: RTMP ingest and Low-Latency HLS delivery
- **APIs**: HTTP REST API (Swagger) and a gRPC service for media transfer

This repository contains Go services plus Docker Compose orchestration for the supporting infrastructure (Postgres, Redis, RabbitMQ, etc.).

## What’s in this repo

- **HTTP API service** (`cmd/http`):
  - Health: `GET /ping`
  - Swagger UI: `GET /swagger/*`
  - Prometheus metrics: `GET /metrics`
  - pprof (debug): `:6060`
  - REST base path: `/api`
  - Swagger spec: `docs/swagger.yaml` / `docs/swagger.json`
- **Livestream service** (Docker profile `livestream`):
  - RTMP ingest + LL-HLS output, configured by `aioz-live.yml`
  - Connect/disconnect webhooks back into the HTTP API
- **Infrastructure**:
  - Postgres (custom image, init SQL in `00_init.sql`)
  - Redis
  - RabbitMQ (management image)

```

## Requirements

- **Go**: 1.22 (see `go.mod`)
- **Docker + Docker Compose**

## Configuration

1) Create `app.env` from the example:

```bash
cp env-example/app.env app.env
```

2) Fill required values (at minimum):
- **Database**: `POSTGRES_*`
- **Redis**: `REDIS_*`
- **RabbitMQ**: `RABBITMQ_*`
- **JWT keys**: `ACCESS_TOKEN_*`, `REFRESH_TOKEN_*` (EdDSA keys in base64 format)
- **Storage paths**: `INPUT_STORAGE_PATH`, `OUTPUT_STORAGE_PATH`

## Run with Docker Compose

This repo uses Compose **profiles**.

### VOD stack (API + Postgres/Redis/RabbitMQ)

```bash
docker compose --profile vod up -d --build
```

### Livestream stack (RTMP ingest + HLS delivery + API dependencies)

```bash
docker compose --profile livestream up -d --build
```

## Useful ports (defaults)

- **HTTP API**: `8080`
  - Swagger UI: `http://localhost:8080/swagger/index.html`
  - Metrics: `http://localhost:8080/metrics`
- **pprof**: `http://localhost:6060/debug/pprof/`
- **gRPC**: `50051`
- **RTMP ingest**: `1935`
- **HLS (LL-HLS)**: `2327`
- **Postgres**: `5437` (host) → `5432` (container)
- **RabbitMQ management**: `15675` (host) → `15672` (container)

## Run locally (without Docker)

Build and run the HTTP API:

```bash
make build
APP_ENV=debug ./bin/api
```

## API documentation

- **Swagger YAML**: `docs/swagger.yaml`
- **Swagger JSON**: `docs/swagger.json`
- **Swagger UI**: `GET /swagger/*` (served by the HTTP API)

Notable API areas include:
- **Auth**: `/auth/*`
- **Livestreams**: `/live_streams*`
- **Playlists**: `/playlists*`
- **Webhooks**: `/webhooks*`

## Notes / repo assumptions

- **Nginx/OpenResty config**: `docker-compose.yml` and `Dockerfile.nginx` reference `nginx.conf` and `nginx/redis_lookup.lua`. These files are expected in the deployment environment (they are not currently present in this repository snapshot).
- **Monitoring configs**: `docker-compose.yml` references `prometheus.yml`, `grafana/datasources.yaml`, and `promtail-config.yaml`. If you enable the `monitor` profile, make sure those files exist.
- **Job/worker system**: `internal/proto/job.proto` describes a job service API used for processing workflows (configured via `JOB_SERVER_HOST` / `JOB_SERVER_PORT`).
