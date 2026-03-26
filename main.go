package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Alertmanager 发送的告警结构
type AlertmanagerWebhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert 单个告警结构
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// 飞书消息结构
type FeishuMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// 配置
type Config struct {
	ListenAddr     string
	FeishuWebhook  string
	MaxMessageSize int64
}

// 从环境变量加载配置
func loadConfig() *Config {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	feishuWebhook := os.Getenv("FEISHU_WEBHOOK")
	//feishuWebhook = "https://open.feishu.cn/open-apis/bot/v2/hook/842123d7-9bae-4a11-a65b-795921637611"

	if feishuWebhook == "" {
		log.Fatal("FEISHU_WEBHOOK environment variable is required")
	}

	return &Config{
		ListenAddr:     listenAddr,
		FeishuWebhook:  feishuWebhook,
		MaxMessageSize: 32 * 1024, // 32KB
	}
}

// 格式化告警为飞书文本消息
func formatAlertToFeishu(webhook *AlertmanagerWebhook) string {
	var builder strings.Builder

	// 状态图标
	statusIcon := "🔴"
	if webhook.Status == "resolved" {
		statusIcon = "🟢"
	}

	// 头部信息
	builder.WriteString(fmt.Sprintf("%s **Alertmanager 告警**\n", statusIcon))
	builder.WriteString(fmt.Sprintf("**状态**: %s\n", webhook.Status))
	builder.WriteString(fmt.Sprintf("**接收器**: %s\n", webhook.Receiver))
	builder.WriteString(fmt.Sprintf("**告警数量**: %d\n\n", len(webhook.Alerts)))

	// 公共标签
	if len(webhook.CommonLabels) > 0 {
		builder.WriteString("**公共标签**:\n")
		for k, v := range webhook.CommonLabels {
			builder.WriteString(fmt.Sprintf("  • %s: %s\n", k, v))
		}
		builder.WriteString("\n")
	}

	// 告警详情
	for i, alert := range webhook.Alerts {
		builder.WriteString(fmt.Sprintf("**告警 %d**:\n", i+1))

		// 告警名称
		if alertName, ok := alert.Labels["alertname"]; ok {
			builder.WriteString(fmt.Sprintf("  • 名称: %s\n", alertName))
		}

		// 严重程度
		if severity, ok := alert.Labels["severity"]; ok {
			builder.WriteString(fmt.Sprintf("  • 级别: %s\n", severity))
		}

		// 告警状态
		alertStatusIcon := "🔴"
		if alert.Status == "resolved" {
			alertStatusIcon = "🟢"
		}
		builder.WriteString(fmt.Sprintf("  • 告警状态: %s %s\n", alertStatusIcon, alert.Status))

		// 摘要信息
		if summary, ok := alert.Annotations["summary"]; ok {
			builder.WriteString(fmt.Sprintf("  • 摘要: %s\n", summary))
		}

		// 描述信息
		if description, ok := alert.Annotations["description"]; ok {
			builder.WriteString(fmt.Sprintf("  • 描述: %s\n", description))
		}

		// 实例信息
		if instance, ok := alert.Labels["instance"]; ok {
			builder.WriteString(fmt.Sprintf("  • 实例: %s\n", instance))
		}

		// 开始时间
		builder.WriteString(fmt.Sprintf("  • 开始时间: %s\n", alert.StartsAt.Format("2006-01-02 15:04:05")))

		// 如果是恢复告警，显示结束时间
		if alert.Status == "resolved" {
			builder.WriteString(fmt.Sprintf("  • 结束时间: %s\n", alert.EndsAt.Format("2006-01-02 15:04:05")))
		}

		// Grafana 链接（如果有）
		if alert.GeneratorURL != "" {
			builder.WriteString(fmt.Sprintf("  • 详情链接: %s\n", alert.GeneratorURL))
		}

		if i < len(webhook.Alerts)-1 {
			builder.WriteString("\n")
		}
	}

	// 添加外部链接
	if webhook.ExternalURL != "" {
		builder.WriteString(fmt.Sprintf("\n**Alertmanager**: %s", webhook.ExternalURL))
	}

	return builder.String()
}

// 发送消息到飞书
func sendToFeishu(webhookURL, message string) error {
	feishuMsg := FeishuMessage{
		MsgType: "text",
	}
	feishuMsg.Content.Text = message

	jsonData, err := json.Marshal(feishuMsg)
	if err != nil {
		return fmt.Errorf("marshal feishu message failed: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("post to feishu failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// 处理 Alertmanager 的 webhook 请求
func handleAlertmanagerWebhook(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受 POST 请求
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 限制请求体大小
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxMessageSize)

		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// 解析 JSON
		var webhook AlertmanagerWebhook
		if err := json.Unmarshal(body, &webhook); err != nil {
			log.Printf("Failed to unmarshal JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 记录接收到的告警
		log.Printf("Received alert: status=%s, alerts=%d", webhook.Status, len(webhook.Alerts))

		// 格式化消息
		message := formatAlertToFeishu(&webhook)

		// 发送到飞书
		if err := sendToFeishu(config.FeishuWebhook, message); err != nil {
			log.Printf("Failed to send to Feishu: %v", err)
			http.Error(w, "Failed to send to Feishu", http.StatusInternalServerError)
			return
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
		})

		log.Printf("Successfully sent alert to Feishu")
	}
}

// 健康检查接口
func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}

func main() {
	// 加载配置
	config := loadConfig()

	// 设置路由
	http.HandleFunc("/webhook", handleAlertmanagerWebhook(config))
	http.HandleFunc("/health", handleHealth())

	// 启动服务
	log.Printf("Starting Alertmanager to Feishu webhook relay on %s", config.ListenAddr)
	log.Printf("Feishu webhook URL: %s", config.FeishuWebhook)

	if err := http.ListenAndServe(config.ListenAddr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
