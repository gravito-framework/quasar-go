# 🌌 Quasar Go Agent

> *"The brightest signal in your infrastructure."*

Quasar is a lightweight, cross-platform monitoring agent that collects system metrics and queue status, sending them to [Zenith](https://github.com/gravito-framework/zenith) for visualization.

## 🚀 Quick Start

### Using Docker (Local Development Example)

```bash
docker run -d \
  -e QUASAR_SERVICE=my-laravel-app \
  -e QUASAR_REDIS_URL=redis://host.docker.internal:6379 \
  gravito/quasar-go-agent:latest
```

> **Note**: `host.docker.internal` is a special DNS name for **local development** that allows the container to reach Redis on your host machine. In production, replace this with your actual Redis host or service name.

### Using Binary

```bash
# Download (Linux/macOS)
curl -sL https://get.gravito.dev/quasar-go | bash

# Run
QUASAR_SERVICE=my-app quasar-go
```

### Building from Source

```bash
go build -o quasar-go ./cmd/quasar

# Run
QUASAR_SERVICE=my-app ./quasar-go
```

### Selecting the Right Redis URL

The `QUASAR_REDIS_URL` depends on your deployment environment:

| Environment | Suggested URL | Description |
|---|---|---|
| **Local (Binary)** | `redis://localhost:6379` | Running both Agent and Redis directly on your machine. |
| **Local (Docker)** | `redis://host.docker.internal:6379` | Running Agent in Docker, but Redis is on your laptop (macOS/Windows). |
| **Docker Compose** | `redis://redis-service-name:6379` | Inside a compose file, use the target service name. |
| **Production/Cloud** | `redis://your-redis-host:6379` | The internal or external DNS of your production Redis (e.g. AWS ElastiCache). |

## 📋 Configuration

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `QUASAR_SERVICE` | ✅ | - | Service name identifier (e.g., `my-api`) |
| `QUASAR_NAME` | ❌ | hostname | Custom node name for the dashboard |
| `QUASAR_TRANSPORT_REDIS_URL` | ❌ | `redis://localhost:6379` | **Transport Layer**: Redis for Zenith (heartbeats & commands) |
| `QUASAR_REDIS_URL` | ❌ | - | Shorthand for `QUASAR_TRANSPORT_REDIS_URL` |
| `QUASAR_MONITOR_REDIS_URL` | ❌ | - | **Monitor Layer**: Redis for your application's queues |
| `QUASAR_INTERVAL` | ❌ | `10` | Heartbeat interval (in seconds) |

## 🔍 Features

### ✅ Phase 1: System Monitoring
- CPU usage (System & Process)
- Memory usage (System & Process RSS)
- Process info (PID, Uptime, Platform)

### ✅ Phase 2: Queue Monitoring
- Redis List queues
- Laravel Queue (Redis driver)
- BullMQ (coming soon)

### ✅ Phase 3: Remote Control
- RETRY_JOB command
- DELETE_JOB command
- Security allowlist

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Your Application                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Laravel App │  │  Node.js    │  │   Legacy System     │ │
│  └─────┬───────┘  └─────────────┘  └──────────┬──────────┘ │
│        │                                       │            │
│        └───────────┬───────────────────────────┘            │
│                    ▼                                        │
│            ┌───────────────┐                                │
│            │ Quasar Agent  │ ◄─── This package              │
│            │    (Go)       │                                │
│            └───────┬───────┘                                │
│                    │                                        │
└────────────────────┼────────────────────────────────────────┘
                     │ Redis Pub/Sub
                     ▼
            ┌───────────────┐
            │    Zenith     │
            │  (Dashboard)  │
            └───────────────┘
```

### 💡 Dual-Redis Design

To ensure performance and security, Quasar uses a separated connection strategy:
- **Transport Redis (Transport Layer)**: This is the agent's "external egress." it usually points to a Redis instance dedicated to Zenith, ensuring monitoring traffic doesn't interfere with your business database.
- **Monitor Redis (Monitor Layer)**: This is the agent's "internal eye." It must point to the Redis instance where your application (e.g., Laravel, BullMQ) stores its queue jobs.

> **Pro Tip**: If Zenith and your application share the same Redis instance, you can set both URLs to the same value.

## 🐳 Docker Compose Example

```yaml
version: '3.8'
services:
  laravel:
    image: my-laravel-app
    depends_on:
      - redis
      - quasar

  quasar:
    image: gravito/quasar-go-agent:latest
    environment:
      QUASAR_SERVICE: my-laravel-app
      QUASAR_TRANSPORT_REDIS_URL: redis://zenith-redis:6379
      QUASAR_MONITOR_REDIS_URL: redis://redis:6379
    depends_on:
      - redis

  redis:
    image: redis:alpine
```

## 🛠️ Development

```bash
# Install dependencies
go mod tidy

# Build
go build -o quasar-go ./cmd/quasar

# Run tests
go test ./...

# Run with debug logging
LOG_LEVEL=debug QUASAR_SERVICE=test ./quasar-go
```

## 📜 License

MIT © Gravito Framework
