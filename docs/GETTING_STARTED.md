# 🚀 Getting Started with GoHop

Welcome to GoHop! This guide will help you get up and running with GoHop in just a few minutes.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [First Steps](#first-steps)
- [Creating Your First Queue](#creating-your-first-queue)
- [Setting Up Retry System](#setting-up-retry-system)
- [Monitoring Queues](#monitoring-queues)
- [Next Steps](#next-steps)

---

## Prerequisites

Before you begin, ensure you have:

- **RabbitMQ** running (local or remote)
  - Management Plugin enabled (port 15672)
  - AMQP port accessible (default: 5672)
- **Go 1.21+** (if installing from source)

### Quick RabbitMQ Setup (Docker)

If you don't have RabbitMQ running, use Docker:

```bash
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=admin \
  -e RABBITMQ_DEFAULT_PASS=admin \
  rabbitmq:3-management
```

Access the Management UI at http://localhost:15672

---

## Installation

### Option 1: Go Install (Recommended)

```bash
go install github.com/davioliveeira/gohop/cmd/gohop@latest
```

### Option 2: From Source

```bash
git clone https://github.com/davioliveeira/gohop.git
cd gohop
make install
```

### Option 3: Pre-built Binaries

Download from [GitHub Releases](https://github.com/davioliveeira/gohop/releases).

### Verify Installation

```bash
gohop --version
```

---

## First Steps

### 1. Configure Your Connection

Run the interactive configuration wizard:

```bash
gohop config init
```

You'll be prompted to enter:
- Host (default: `localhost`)
- Port (default: `5672`)
- Management Port (default: `15672`)
- Username
- Password
- VHost (default: `/`)

#### Alternative: Environment Variables

Create a `.env` file in your project:

```env
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_USER=admin
RABBITMQ_PASSWORD=admin
RABBITMQ_VHOST=/
```

#### Alternative: Direct URL

You can also configure using a connection URL:

```bash
gohop config init
# Choose "URL" option and paste:
# amqp://admin:admin@localhost:5672/
```

### 2. Test Your Connection

```bash
gohop config test
```

Expected output:
```
✅ Conexão AMQP: OK
✅ Management API: OK
```

### 3. Launch Interactive Mode

```bash
gohop
```

This opens the beautiful TUI menu where you can perform all operations.

---

## Creating Your First Queue

### Interactive Mode

1. Run `gohop`
2. Select **"➕ Criar Fila"**
3. Fill in the form:
   - Queue name: `my-first-queue`
   - Type: `classic` or `quorum`
   - With retry: `yes` (recommended)

### CLI Mode

```bash
# Simple queue
gohop queue create my-first-queue

# Queue with retry system
gohop queue create my-first-queue --with-retry --max-retries 3 --retry-delay 5000
```

---

## Setting Up Retry System

The retry system automatically handles failed messages:

```
Message Rejected → Wait Queue (5s delay) → Retry
                                        ↓
                           After 3 retries → DLQ
```

### Components Created

When you enable retry for `my-queue`, GoHop creates:

| Component | Purpose |
|-----------|---------|
| `my-queue` | Main queue |
| `my-queue.wait` | Holds messages during retry delay |
| `my-queue.dlq` | Dead Letter Queue for failed messages |
| `my-queue.retry-exchange` | Routes retries back to main queue |
| `my-queue.wait-exchange` | Routes to wait queue |

### Configure Retry for Existing Queue

```bash
# Via interactive menu
gohop
# Select "🔧 Reconfigurar Fila"

# Via CLI
gohop retry setup my-existing-queue --max-retries 5 --retry-delay 10000
```

### Check Retry Status

```bash
gohop retry status my-queue
```

---

## Monitoring Queues

### Single Queue Dashboard

```bash
gohop monitor my-queue
```

Features:
- Real-time message count
- Consumer status
- Retry system health
- Progress bars
- Auto-refresh every 20s

### Multi-Queue Dashboard

1. Run `gohop`
2. Select **"📊 Monitorar Fila"**
3. Choose **"Monitorar múltiplas filas"**
4. Select queues with `Space`
5. Press `Enter` to start

---

## Common Operations

### List All Queues

```bash
gohop queue list
```

### Get Queue Details

```bash
gohop queue status my-queue
```

### Purge Messages

```bash
gohop queue purge my-queue
```

### Delete Queue

```bash
gohop queue delete my-queue
```

---

## Configuration Profiles

Use profiles to manage multiple RabbitMQ environments:

```bash
# Create profiles
gohop config init --profile production
gohop config init --profile staging

# Use a specific profile
gohop --profile production queue list

# List profiles
gohop config list
```

---

## Troubleshooting

### Connection Refused

```
Error: dial tcp: connection refused
```

**Solution**: Check if RabbitMQ is running and ports are accessible.

### Access Denied

```
Error: Exception (403) Reason: "no access to this vhost"
```

**Solution**: Verify username/password and VHost permissions.

### Management API Error

```
Error: Management API connection failed
```

**Solution**: Ensure the Management Plugin is enabled and port 15672 is accessible.

---

## Next Steps

- 📖 Read the [full documentation](./ARCHITECTURE.md)
- 🔧 Explore [advanced configuration](./CONFIGURATION.md)
- 🤝 [Contribute](../CONTRIBUTING.md) to the project

---

<p align="center">
  <strong>Need help?</strong> Open an <a href="https://github.com/davioliveeira/gohop/issues">issue</a> on GitHub.
</p>
