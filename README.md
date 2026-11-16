# Rate Limiter

## Quick Start (Docker Compose)

The whole setup can be run from the root `docker-compose.yaml`:

```bash
docker compose up --build -d
```

The rate limiter is configured through environment variables in `docker-compose.yaml`:

```yaml
RATE_LIMIT_LIMIT: 5              # max requests allowed per window
RATE_LIMIT_WINDOW: 1m            # window size
RATE_LIMIT_BLOCK_DURATION: 30s   # how long a client stays blocked after exceeding the limit
```

After changing these values, re-run `docker compose up --build -d` to apply them.

The sections below describe running each component manually.

## Run Redis

```bash
cd redis
docker compose up -d
```

To enter Redis using the CLI:

```bash
docker exec -it rate-limiter-redis redis-cli
```

## Start the Go Rate-Limit Service

```bash
cd ~/KWAAP/rateLimiting/rate-limiter
go run ./cmd/server
```

## Start the Node.js Backend

From the backend directory:

```bash
cd backend
npm install
npm start
```

## Run Envoy Locally

### 3. Validate the Configuration

Run this from the `rateLimiting/envoy` directory:

```bash
docker run --rm \
  --network host \
  -v "$PWD/envoy.yaml:/etc/envoy/envoy.yaml:ro" \
  envoyproxy/envoy:v1.40-latest \
  --mode validate \
  -c /etc/envoy/envoy.yaml
```

Expected result:

```text
configuration '/etc/envoy/envoy.yaml' OK
```

### 4. If Validation Succeeds

Start Envoy:

```bash
docker run --rm \
  --name rate-limit-envoy \
  --network host \
  -v "$PWD/envoy.yaml:/etc/envoy/envoy.yaml:ro" \
  envoyproxy/envoy:v1.40-latest \
  -c /etc/envoy/envoy.yaml \
  --log-level info
```