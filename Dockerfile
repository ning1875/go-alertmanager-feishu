FROM registry.cn-beijing.aliyuncs.com/ning1875_haiwai_image/golang:1.25.1 as builder
WORKDIR /app
#COPY go.mod ./
#COPY go.sum ./


#RUN go env -w GOPROXY=https://goproxy.cn,direct ; go mod download
COPY . .

RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64 go build -o go-alertmanager-feishu main.go


# 第二阶段：运行
FROM registry.cn-beijing.aliyuncs.com/ning1875_haiwai_image/alpine:latest

# 安装必要的运行时依赖

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/go-alertmanager-feishu .


FROM registry.cn-beijing.aliyuncs.com/ning1875_haiwai_image/alpine as runner
COPY --from=builder /app/go-alertmanager-feishu .
ENTRYPOINT [ "./go-alertmanager-feishu" ]
