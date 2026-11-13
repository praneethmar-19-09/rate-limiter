# Rate Limiter

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