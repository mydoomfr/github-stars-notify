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
- 🔔 **Discord & Slack notifications** with rich embeds
- 📊 **Prometheus metrics** built-in with Grafana dashboard
- ⚡ **GitHub Rate limit aware** and optimized
- 🐳 **Docker ready** with multi-platform support

## 🎛️ Configuration

```yaml
# config.yaml
repositories:
  - owner: "your-org"
    repo: "awesome-project"

notifications:
  discord:
    enabled: false
    webhook_url: "https://discord.com/api/webhooks/..."
  
  slack:
    enabled: false
    webhook_url: "https://hooks.slack.com/services/..."
    channel: "#github-stars"

settings:
  check_interval_minutes: 60
```

## 📄 License

Apache 2.0 License - see [LICENSE](LICENSE) file for details.

---

<div align="center">

**⭐ If you find this project useful, please star it! LOL. ⭐**

</div> 