# Alertmanager to Feishu Webhook Relay

一个轻量级的 Golang 服务，用于接收 Prometheus Alertmanager 的告警，并将其转发到飞书（Feishu/Lark）群聊机器人。

## 功能特性

- 🚀 **轻量高效**：基于 Golang 实现，资源占用低，启动快速
- 📝 **格式化消息**：将 Alertmanager 告警转换为易读的飞书文本消息
- 🔔 **支持告警恢复**：区分告警触发和恢复状态，使用不同图标标识
- 🏥 **健康检查**：提供 `/health` 接口用于健康状态监控
- 🔒 **安全限制**：请求体大小限制，防止恶意请求
- 📦 **容器化支持**：提供 Dockerfile，便于容器化部署
- ⚙️ **配置灵活**：通过环境变量配置，易于集成到容器编排平台

## 快速开始

### 环境要求

- Go 1.21 或更高版本（仅编译时需要）
- Docker（可选，用于容器化部署）

### 安装与运行

#### 方式一：直接运行

1. 克隆代码：
```bash
git clone https://github.com/yourusername/alertmanager-feishu.git
cd alertmanager-feishu