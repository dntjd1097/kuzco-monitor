# Kuzco Monitor

A monitoring service for Kuzco workers with Telegram notifications. This service provides real-time monitoring of worker status, token usage, and GPU utilization with automated reports.

## 🌟 Features

### Real-time Monitoring

-   Worker status changes (1-minute intervals)
-   Instance status tracking
-   GPU utilization monitoring
-   IP address change detection

### Automated Reports

-   Hourly token reports (at :05 of every hour)
-   Daily summary reports (UTC 00:00)
-   Token usage statistics
-   Worker performance metrics

### Telegram Integration

-   Separate threads for different notification types
-   Interactive commands
-   Formatted messages

## 📋 Requirements

-   Docker
-   Docker Compose
-   Telegram Bot Token
-   Kuzco Account

## 🚀 Quick Start

1. **Clone the repository**

    ```bash
    git clone https://github.com/yourusername/kuzco-monitor.git
    cd kuzco-monitor
    ```

2. **Create configuration file**

    ```bash
    cp config.yaml.example config.yaml
    ```

3. **Configure config.yaml**

    ```yaml
    kuzco:
        id: 'your-email@example.com'
        password: 'your-password'
    telegram:
        token: 'your-telegram-bot-token'
        chat_id: 'your-chat-id'
        threads:
            daily: 5 # Daily report thread
            hourly: 6 # Hourly report thread
            error: 7 # Error message thread
            status: 8 # Status message thread
    ```

4. **Start the service**

    ```bash
    docker-compose up -d
    ```

## ⚙️ Telegram Setup

1. Create a bot through BotFather
2. Start conversation with bot (`/start` command)
3. Create a group and add the bot
4. Create threads and configure thread IDs

## 🤖 Telegram Commands

| Command   | Description                | Thread |
| --------- | -------------------------- | ------ |
| `/status` | View current worker status | Status |
| `/report` | Generate full report       | Daily  |
| `/help`   | List available commands    | Status |

## 📊 Report Types

### Hourly Report

-   Token usage changes
-   Generation counts
-   Worker-specific metrics
-   Per-instance performance

### Daily Report

-   24-hour summary
-   Global token statistics
-   Worker performance analysis
-   Instance utilization

## 🔍 Monitoring Details

### Worker Status (1-minute intervals)

-   Instance count changes
-   Status changes (Running/Initializing)
-   IP address changes
-   Worker additions/removals

### Performance Metrics

-   Tokens per instance
-   Generations per instance
-   Token share percentage
-   GPU utilization

## 🛠️ Development Environment

-   Go 1.21+
-   Docker
-   Docker Compose

## 📝 Viewing Logs

```bash
# View all logs
docker-compose logs --tail=100 -f

# View logs since specific time
docker-compose logs --since "2024-01-01T00:00:00" -f
```

## 🔒 Security

-   Configuration file contains sensitive information, ensure proper permissions
-   `.gitignore` prevents config files from being committed
-   Docker volumes manage configuration securely

## 🔍 Monitoring Details

### Worker Metrics

-   Real-time GPU utilization
-   Memory usage
-   Power consumption
-   Temperature monitoring

### Alert Types

-   Worker status changes
-   Instance initialization/termination
-   Performance anomalies
-   Error conditions

## 📊 Report Examples

### Status Update

```
📊 Current Status
Online Workers: 5
Total Instances: 12
Running: 10
Initializing: 2
```

### Hourly Report

```
⏰ Hourly Token Report
Global Changes (Last Hour):
- Tokens: 1,234,567 → 1,345,678 (Δ111,111)
Worker Changes:
- Worker1: 50,000 → 55,000 (Δ5,000)
- Worker2: 45,000 → 48,000 (Δ3,000)
```

## 🛠️ Maintenance

### Service Management

```bash
# Restart service
docker-compose restart

# View service status
docker-compose ps

# Update service
docker-compose pull
docker-compose up -d
```

### Backup Configuration

```bash
# Backup config
cp config.yaml config.yaml.backup

# Restore config
cp config.yaml.backup config.yaml
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📄 License

MIT License

## 📧 Support

For support, please create an issue in the GitHub repository or contact the maintainers.

---

**Note:** Keep your configuration file secure and never commit sensitive information to the repository.
