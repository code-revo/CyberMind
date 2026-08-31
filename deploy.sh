#!/usr/bin/env bash
#
# CyberMind 服务器自动部署脚本
# 由 GitHub Actions(或手动)在服务器上执行:
#   1. git pull 拉取最新前后端代码
#   2. 编译 Go 后端二进制
#   3. 平滑重启 cybermind 进程(前后端由二进制统一托管,所以一起更新)
#
# 用法: ./deploy.sh
# 可选环境变量:
#   DEPLOY_START_ARGS  覆盖启动参数(默认 "-config config.yaml")
#                      例如 ./cybermind 原启动参数,会自动保留
set -euo pipefail

# 非交互式 SSH(如 GitHub Actions)可能不在 PATH 里,显式加上 Go
export PATH="/usr/local/go/bin:${PATH:-}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

cd "$(dirname "$0")"

echo "==> [1/4] 拉取最新代码"
# 先尝试快进拉取;若因 force-push 导致历史分叉(首次部署或历史重写),
# 则重置到远端 master(本地未跟踪的 config.yaml / data/ 不受影响)。
if ! git pull --ff-only origin master 2>/dev/null; then
  echo "  ⚠️  本地与远端历史分叉,重置到远端 master"
  git fetch origin master
  git reset --hard FETCH_HEAD
fi
echo "  当前版本: $(git rev-parse --short HEAD)"

echo "==> [2/4] 检查 Go 工具链"
if ! command -v go >/dev/null 2>&1; then
  echo "❌ 服务器未安装 Go,请先安装(编译后端需要):" >&2
  echo "   Ubuntu/Debian:  sudo apt-get install golang-go" >&2
  echo "   CentOS/RHEL:    sudo yum install golang" >&2
  echo "   或参考:         https://go.dev/dl/" >&2
  exit 1
fi
go version

echo "==> [3/4] 编译后端"
go build -trimpath -ldflags="-s -w" -o cybermind.new cmd/server/main.go
mv -f cybermind.new cybermind
chmod +x cybermind
echo "    编译完成: $(ls -lh cybermind | awk '{print $5}')"

echo "==> [4/4] 重启服务"

restart_via_systemd() {
  sudo systemctl restart cybermind
  sleep 1
  if systemctl is-active --quiet cybermind; then
    echo "✅ 已通过 systemd 重启 cybermind"
    systemctl status cybermind --no-pager | head -12
    return 0
  fi
  echo "⚠️  systemd 重启后未存活,回退到 nohup 方式" >&2
  return 1
}

# 方式A: 若存在 systemd 服务则优先使用
if systemctl list-unit-files 2>/dev/null | grep -q '^cybermind\.service'; then
  if restart_via_systemd; then
    exit 0
  fi
fi

# 方式B: nohup 后台进程(保留原启动参数重启)
SUDO=""
PID="$(pgrep -f 'cybermind' | head -1 || true)"
if [ -n "$PID" ]; then
  # 若进程属于其他用户,需要用 sudo 才能 kill
  if [ "$(ps -o user= -p "$PID" | tr -d ' ')" != "$(id -un)" ]; then
    SUDO="sudo"
  fi
fi

# 尽量从 /proc/PID/cmdline 还原原启动参数,保留 -config/--https 等
RESTART_ARGS="${DEPLOY_START_ARGS:-}"
if [ -z "$RESTART_ARGS" ] && [ -n "$PID" ] && [ -r "/proc/$PID/cmdline" ]; then
  RESTART_ARGS="$(tr '\0' ' ' < "/proc/$PID/cmdline" | sed 's#^.*[ /]cybermind[^ ]* ##')"
fi
RESTART_ARGS="${RESTART_ARGS:- -config config.yaml}"

if [ -n "$PID" ]; then
  echo "    停止旧进程 PID=$PID"
  $SUDO kill "$PID" 2>/dev/null || true
  for _ in $(seq 1 20); do
    if ! kill -0 "$PID" 2>/dev/null; then break; fi
    sleep 0.5
  done
  $SUDO kill -9 "$PID" 2>/dev/null || true
fi

# 用 nohup 重新启动(输出到 cybermind.log)
# shellcheck disable=SC2086
nohup ./cybermind $RESTART_ARGS > cybermind.log 2>&1 &
NEW_PID=$!
echo "    已启动 PID=$NEW_PID,参数:${RESTART_ARGS}"
sleep 2

if kill -0 "$NEW_PID" 2>/dev/null; then
  echo "✅ 重启成功,进程存活 (PID=$NEW_PID)"
  echo "    日志: cybermind.log"
else
  echo "⚠️  进程未存活,最近日志:" >&2
  tail -n 20 cybermind.log >&2
  exit 1
fi
