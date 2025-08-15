# P2P Chess Backend

## Setup
- Install Postgres, Redis, coturn
- Set env: DB_DSN, REDIS_URL, JWT_KEYS, TURN_SECRET, etc.
- Run migrations: go run cmd/migrate/main.go (if separate)
- Start: go run cmd/api/main.go

## Ports
- API: 8080 (or 443 with nginx)
- coturn: 3478/5349

## SLOs
- Uptime 99.9%
- Append latency <500ms
- Matchmaking <5s

// TODO: Dashboards, alerts