# CyberMind · 网络空间自主决策引擎

> 具备自主决策能力的通用网络安全智能体

CyberMind 是一个面向网络安全实战场景的**自主决策智能体**，能够准确理解自然语言、压缩文件、接口文档等多种格式的安全任务目标，自动分解任务、生成可操作执行计划，并在资产探测、应用层漏洞挖掘、应急响应、攻击路径规划等场景中动态调整策略、实时执行，且关键决策**可解释、可审计**。

## 核心能力

| 能力 | 说明 |
|------|------|
| **自主决策** | 理解自然语言 / 压缩文件 / 接口文档 → 任务分解 → 生成可执行计划 |
| **多场景执行** | 资产探测、应用层漏洞挖掘、应急响应、攻击路径规划，动态态势分析、目标设定、路径规划、实时自动调整 |
| **执行高可靠** | 决策可解释、关键决策可审计，复杂对抗环境稳定运行，行为可复现 |

## 技术栈

- **语言**：Go 1.25+（纯 Go 编译，无 CGO 依赖）
- **Web**：Gin + 原生前端（SSE 流式 / WebSocket 终端）
- **多智能体**：CloudWeGo Eino（ADK 单智能体 + 多智能体编排）
- **工具协议**：Model Context Protocol（MCP，含外部 MCP 服务器接入）
- **存储**：SQLite（`modernc.org/sqlite`，纯 Go 驱动）
- **大模型**：支持国产大模型（DeepSeek / 通义千问 等 OpenAI 兼容通道，可配置多通道主备切换）

## 目录结构

```
├── cmd/server/            # 服务入口
├── internal/              # 核心实现（agent / multiagent / mcp / hitl / knowledge / audit …）
├── tools/                 # 90+ 安全工具 YAML 配方（nmap/sqlmap/fscan/metasploit/ghidra …）
├── skills/                # 25 个攻击技能包（web/二进制/云/区块链/CTF/后渗透 …）
├── agents/                # 16 个专项子代理提示词
├── roles/                 # 14 个角色定义
├── knowledge_base/        # 知识库（RAG 检索增强）
├── mcp-servers/           # MCP 服务器实现与示例
├── web/                   # 前端（对话式控制台）
├── config.example.yaml    # 配置模板（提交用脱敏版本）
└── run.sh                 # Linux / macOS 一键部署启动脚本
```

## 快速开始

### Linux / macOS

```bash
./run.sh            # 一键：检查环境 → 装依赖 → 编译 → 启动
./run.sh --http     # 以纯 HTTP 方式启动
```

访问 `http://127.0.0.1:8080`（`run.sh` 默认 HTTPS 自签证书）。

### Windows

```bash
deploy-windows.bat  # 一键：检查 Go/Python → 编译 → 生成配置 → 启动
```

### 手动构建

```bash
go mod download
go build -o cybermind cmd/server/main.go
./cybermind -config config.yaml
```

## 配置

首次启动会根据 `config.example.yaml` 自动生成 `config.yaml`。在 `ai.channels` 中配置国产大模型通道（如 DeepSeek），`ai.default_channel` 指定默认通道。

```yaml
ai:
  default_channel: deepseek-main
  channels:
    deepseek-main:
      provider: openai_compatible
      api_key: sk-xxxxxxxx
      base_url: https://api.deepseek.com/v1
      model: deepseek-chat
```

> 提交 / 传播前请务必将真实 `api_key` 脱敏（本项目 `.gitignore` 已排除 `config.yaml`，提交用 `config.example.yaml`）。

## 许可与来源声明

本项目基于 Apache License 2.0 开源项目 [CyberStrikeAI](https://github.com/Ed1s0nZ/CyberStrikeAI)（1.7.17）深度定制，在其基础上完成了**自主决策引擎改造、多智能体编排增强、免登录安全架构、国产大模型适配、品牌化前端重构**等原创增量工作。Apache 2.0 允许在注明来源的前提下进行二次开发与再分发。详见 `LICENSE` 与《原创性及保密性声明函》。
