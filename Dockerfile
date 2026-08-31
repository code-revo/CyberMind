# ============ 构建阶段 ============
FROM golang:1.25-alpine AS builder

# 国内依赖加速 + 纯静态编译（本项目无 CGO，可关闭）
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /src

# 先只复制依赖清单，命中 Docker 层缓存（源码改动不重下依赖）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译出静态二进制
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/cybermind cmd/server/main.go

# ============ 运行阶段 ============
FROM alpine:3.20

# 可选：若需要运行 Python 工具配方（api-fuzzer / dnslog / impacket 等），
# 取消下面注释以安装 Python 运行时与依赖（部分包如 angr 需额外编译工具链）。
# RUN apk add --no-cache python3 py3-pip gcc musl-dev libffi-dev && \
#     pip3 install --no-cache-dir --break-system-packages -r /requirements.txt

WORKDIR /app

# 二进制
COPY --from=builder /out/cybermind /app/cybermind

# 运行时资源目录（前端模板/静态 + 工具/技能/角色/子代理/知识库/MCP）
COPY --from=builder /src/web ./web
COPY --from=builder /src/tools ./tools
COPY --from=builder /src/skills ./skills
COPY --from=builder /src/agents ./agents
COPY --from=builder /src/roles ./roles
COPY --from=builder /src/knowledge_base ./knowledge_base
COPY --from=builder /src/mcp-servers ./mcp-servers
COPY --from=builder /src/config.example.yaml ./config.example.yaml
COPY --from=builder /src/requirements.txt ./requirements.txt

# SQLite 数据目录（通过卷持久化）
RUN mkdir -p /app/data
VOLUME /app/data

EXPOSE 8080

ENTRYPOINT ["/app/cybermind"]
CMD ["-config", "config.yaml"]
