# ⭐ GitHub Stars Notify

> **Monitor your GitHub repositories and get notified when they receive new stars!**

<div align="center">

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![License](https://img.shields.io/badge/Apache-2.0-green?style=for-the-badge)

</div>

---

## 🚀 What is it?

A **lightweight Go service** that watches your GitHub repositories and sends beautiful notifications whenever they receive new stars.

## ✨ Features

- 🌟 **Real-time star monitoring** for multiple repositories
- 🔔 **Discord & Slack notifications** with rich embeds (more coming soon)
- 📊 **Prometheus metrics** built-in with Grafana dashboard
- ⚡ **GitHub Rate limit aware** and optimized
- 🐳 **Container image ready** with multi-platform support

## 🐳 Docker

```
docker run --rm -it -p 9090:9090 -v ./config.yaml:/app/config.yaml github-stars-notify:latest
```

## 🎛️ Configuration

### Basic Configuration

Create a `config.yaml` file (copy from `config.yaml.example`) and mount it to `/app/config.yaml`:

```yaml
# config.yaml
repositories:
  - owner: "your-org"
    repo: "awesome-project"

notifications:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/..."
  
  slack:
    enabled: false
    webhook_url: "https://hooks.slack.com/services/..."
    channel: "#github-stars"
```

### Full Configuration Options

```yaml
# All settings with defaults shown
repositories:
  - owner: "your-org"
    repo: "awesome-project"

settings:
  check_interval_minutes: 60  # Default: 60

github:
  token: ""              # Default: "" (optional but recommended)
  timeout_seconds: 30    # Default: 30

server:
  port: 9090            # Default: 9090
  host: "localhost"     # Default: "localhost"
  read_timeout_seconds: 30   # Default: 30
  write_timeout_seconds: 30  # Default: 30

storage:
  type: "file"          # Default: "file"
  path: "./data"        # Default: "./data"

logging:
  level: "info"         # Default: "info" (debug, info, warn, error)
  format: "text"        # Default: "text" (text, json)

notifications:
  discord:
    enabled: false
    webhook_url: ""
  slack:
    enabled: false
    webhook_url: ""
    channel: ""         # Optional
```

## 🌍 Environment Variables

You can override any configuration value using environment variables:

### Core Settings
| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `GITHUB_TOKEN` | GitHub API token | `ghp_xxxxxxxxxxxx` |
| `CHECK_INTERVAL_MINUTES` | Check interval in minutes | `30` |

### Server Configuration
| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `9090` |
| `SERVER_HOST` | HTTP server host | `localhost` |

### Discord Notifications
| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `DISCORD_ENABLED` | Enable Discord notifications | `true` |
| `DISCORD_WEBHOOK_URL` | Discord webhook URL | `https://discord.com/api/webhooks/...` |

### Slack Notifications
| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `SLACK_ENABLED` | Enable Slack notifications | `true` |
| `SLACK_WEBHOOK_URL` | Slack webhook URL | `https://hooks.slack.com/services/...` |
| `SLACK_CHANNEL` | Slack channel override | `#github-stars` |

### Storage & Logging
| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `STORAGE_PATH` | Storage directory path | `./data` |
| `LOG_LEVEL` | Logging level | `info` |
| `LOG_FORMAT` | Log format (text/json) | `text` |

## 📄 License

Apache 2.0 License - see [LICENSE](LICENSE) file for details.

---

<div align="center">

**⭐ If you find this project useful, please star it! ⭐**

</div> 