<p align="center">
  <img src="https://raw.githubusercontent.com/davioliveeira/gohop/main/.github/logo.png" alt="GoHop Logo" width="200">
</p>

<h1 align="center">🐰 GoHop</h1>

<p align="center">
  <strong>A beautiful and powerful CLI for RabbitMQ management</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#usage">Usage</a> •
  <a href="#screenshots">Screenshots</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
  <img src="https://img.shields.io/badge/RabbitMQ-3.x-FF6600?logo=rabbitmq" alt="RabbitMQ">
  <img src="https://goreportcard.com/badge/github.com/davioliveeira/gohop" alt="Go Report Card">
</p>

---

## ✨ Features

- 🎨 **Beautiful TUI** - Interactive terminal interface powered by [Charm](https://charm.sh)
- 📊 **Real-time Monitoring** - Live dashboard for queue metrics
- 🔄 **Retry System** - Built-in retry logic with Dead Letter Queues
- ⚡ **Queue Management** - Create, delete, purge, and reconfigure queues
- 🔧 **Easy Configuration** - Interactive setup wizard
- 📈 **Multi-queue Dashboard** - Monitor multiple queues simultaneously
- 🎯 **Zero Message Loss** - Safe queue reconfiguration preserving all messages

## 📦 Installation

### Using Go

```bash
go install github.com/davioliveeira/gohop/cmd/gohop@latest
```

### From Source

```bash
git clone https://github.com/davioliveeira/gohop.git
cd gohop
make build
```

### Pre-built Binaries

Download from [Releases](https://github.com/davioliveeira/gohop/releases).

## 🚀 Quick Start

### 1. Configure Connection

```bash
gohop config init
```

Or create a `.env` file:

```env
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_VHOST=/
```

### 2. Run Interactive Mode

```bash
gohop
```

This opens the beautiful interactive menu where you can:
- Create queues with retry/DLQ
- Monitor queues in real-time
- Manage existing queues
- Reconfigure queues without losing messages

## 📖 Usage

### Interactive Mode (Recommended)

```bash
gohop
```

### CLI Commands

```bash
# Configuration
gohop config init          # Interactive setup
gohop config test          # Test connection
gohop config view          # Show current config

# Queue Management
gohop queue list           # List all queues
gohop queue create <name>  # Create a queue
gohop queue delete <name>  # Delete a queue
gohop queue purge <name>   # Purge messages
gohop queue status <name>  # Queue details

# Retry System
gohop retry setup <name>   # Setup retry + DLQ
gohop retry status <name>  # Check retry system

# Monitoring
gohop monitor <name>       # Real-time dashboard
```

## 🎯 Retry System Architecture

GoHop implements a robust retry system with Dead Letter Queues:

```
┌─────────────┐    reject     ┌──────────────┐
│  Main Queue │──────────────▶│ Wait Exchange │
└─────────────┘               └──────────────┘
       ▲                             │
       │                             ▼
       │                      ┌─────────────┐
       │        TTL expires   │ Wait Queue  │
       │◀─────────────────────│  (5s delay) │
       │                      └─────────────┘
       │                             │
       │   retry < max               │ retry >= max
       │◀────────────────────────────┤
                                     ▼
                              ┌─────────────┐
                              │     DLQ     │
                              │ (Dead Letter)│
                              └─────────────┘
```

**Benefits:**
- ✅ No infinite loops
- ✅ Configurable retry count
- ✅ Configurable delay between retries
- ✅ Failed messages preserved in DLQ
- ✅ Easy reprocessing from DLQ

## 🖼️ Screenshots

### Main Menu
```
   ╔═══════════════════════════════════════════════════════════════╗
   ║                                                               ║
   ║   🐰  G O H O P  -  R a b b i t M Q   C L I                  ║
   ║                                                               ║
   ╚═══════════════════════════════════════════════════════════════╝

   Gerencie suas filas RabbitMQ com estilo e eficiência.

   ➕  Criar Fila
   📋  Listar Filas  
   📊  Monitorar Fila
   🔧  Reconfigurar Fila
   ...
```

### Queue Dashboard
```
╭──────────────────────────────────────────────────────────────────╮
│  📊 DASHBOARD: my-queue                                          │
│                                                                  │
│  Messages Ready:    ██████████░░░░░░░░░░  150                   │
│  Unacknowledged:    ░░░░░░░░░░░░░░░░░░░░  0                     │
│  Consumers:         2 active                                     │
│                                                                  │
│  🔄 Retry System: ✓ Active                                      │
│  📭 DLQ Messages: 0                                              │
╰──────────────────────────────────────────────────────────────────╯
```

## 🛠️ Development

### Requirements

- Go 1.21+
- Docker (for integration tests)
- Make

### Build

```bash
make build
```

### Test

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# All tests with coverage
make test-coverage
```

### Project Structure

```
gohop/
├── cmd/
│   ├── gohop/          # Main entry point
│   └── commands/       # CLI commands (Cobra)
├── internal/
│   ├── config/         # Configuration management
│   ├── rabbitmq/       # RabbitMQ client & management API
│   ├── retry/          # Retry system logic
│   └── ui/             # TUI components (Bubble Tea)
├── scripts/            # Helper scripts
├── Makefile
└── README.md
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Charm](https://charm.sh) - For the amazing TUI libraries
- [RabbitMQ](https://www.rabbitmq.com) - The message broker
- [Cobra](https://github.com/spf13/cobra) - CLI framework

---

<p align="center">
  Made with ❤️ and Go
</p>
