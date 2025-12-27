#!/usr/bin/env bash
set -euo pipefail

# Luckfox Pico-Ultra 默认参数（可通过环境变量覆盖）
TARGET_HOST="${TARGET_HOST:-192.168.2.221}"
TARGET_USER="${TARGET_USER:-root}"
TARGET_PORT="${TARGET_PORT:-22}"
TARGET_PATH="${TARGET_PATH:-/root/totoro-device}"
TARGET_PASS="${TARGET_PASS:-}"
# 设备上测试端口：避免占用固件自带的 80 端口服务导致异常/重启
NWCT_HTTP_PORT="${NWCT_HTTP_PORT:-18080}"

# 运行参数：默认禁用 display，避免设备无 /dev/fb0 或权限问题导致启动失败
RUN_ARGS="${RUN_ARGS:--display=false}"

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SSHPASS_PREFIX=()
if [[ -n "${TARGET_PASS}" ]]; then
  if ! command -v sshpass >/dev/null 2>&1; then
    echo "已设置 TARGET_PASS 但本机缺少 sshpass；请先安装：brew install sshpass" >&2
    exit 1
  fi
  SSHPASS_PREFIX=(sshpass -p "${TARGET_PASS}")
fi

ssh_cmd() {
  "${SSHPASS_PREFIX[@]}" ssh -p "${TARGET_PORT}" -o StrictHostKeyChecking=accept-new "${TARGET_USER}@${TARGET_HOST}" "$@"
}

scp_cmd() {
  "${SSHPASS_PREFIX[@]}" scp -P "${TARGET_PORT}" -o StrictHostKeyChecking=accept-new "$@"
}

echo "[1/4] 检测开发板架构..."
ARCH_RAW="$(ssh_cmd uname -m | tr -d '\r' | tr '[:upper:]' '[:lower:]' || true)"
if [[ -z "${ARCH_RAW}" ]]; then
  echo "无法通过 SSH 获取 uname -m；请确认能 ssh ${TARGET_USER}@${TARGET_HOST} 登录。" >&2
  exit 1
fi
echo "板子 uname -m: ${ARCH_RAW}"

GOOS=linux
GOARCH=""
GOARM=""

case "${ARCH_RAW}" in
  armv7l|armv7*|armhf)
    GOARCH=arm
    GOARM=7
    ;;
  armv6l|armv6*)
    GOARCH=arm
    GOARM=6
    ;;
  aarch64|arm64)
    GOARCH=arm64
    ;;
  *)
    echo "不支持/未知架构: ${ARCH_RAW}（请把 uname -m 输出发我，我来补映射）" >&2
    exit 1
    ;;
esac

echo "[2/4] 本机交叉编译（Buildroot 友好：CGO=0）..."
OUT="${PROJECT_DIR}/bin/totoro-device_${GOOS}_${GOARCH}${GOARM:+v${GOARM}}"
mkdir -p "${PROJECT_DIR}/bin"

pushd "${PROJECT_DIR}" >/dev/null
  env \
    CGO_ENABLED=0 \
    GOOS="${GOOS}" \
    GOARCH="${GOARCH}" \
    ${GOARM:+GOARM="${GOARM}"} \
    go build -trimpath -ldflags "-s -w" -o "${OUT}" .
popd >/dev/null

echo "编译产物: ${OUT}"

echo "[3/4] 上传到开发板: ${TARGET_USER}@${TARGET_HOST}:${TARGET_PATH}"
scp_cmd "${OUT}" "${TARGET_USER}@${TARGET_HOST}:${TARGET_PATH}"
ssh_cmd "chmod +x '${TARGET_PATH}'"

echo "[4/4] 远程运行测试（前台运行，Ctrl+C 可中断；或你也可以改用 nohup 后台）..."
echo "执行（后台）：${TARGET_PATH} ${RUN_ARGS}"
ssh_cmd "sh -lc '
set -e

# 使用 /tmp 下的“测试配置”，避免改动系统 /etc 配置、避免占用 80 端口
mkdir -p /tmp/nwct /tmp/nwct/log
cat >/tmp/nwct/config.json <<EOF
{
  \"initialized\": false,
  \"device\": {\"id\": \"DEV001\", \"name\": \"Luckfox\"},
  \"network\": {\"interface\": \"eth0\", \"ip_mode\": \"dhcp\"},
  \"scanner\": {\"auto_scan\": false, \"scan_interval\": 300, \"timeout\": 30, \"concurrency\": 2},
  \"server\": {\"port\": ${NWCT_HTTP_PORT}, \"host\": \"0.0.0.0\"},
  \"database\": {\"path\": \"/tmp/nwct/devices.db\"},
  \"auth\": {\"password_hash\": \"\"}
}
EOF

export NWCT_CONFIG_PATH=/tmp/nwct/config.json
export NWCT_DB_PATH=/tmp/nwct/devices.db
export NWCT_LOG_DIR=/tmp/nwct/log

# 尝试停止上一次运行（如果有）
if [ -f /tmp/nwct.pid ]; then
  oldpid=\$(cat /tmp/nwct.pid 2>/dev/null || true)
  if [ -n \"\${oldpid}\" ] && kill -0 \"\${oldpid}\" 2>/dev/null; then
    kill \"\${oldpid}\" 2>/dev/null || true
    sleep 1
  fi
fi

nohup \"${TARGET_PATH}\" ${RUN_ARGS} >/tmp/nwct.out 2>&1 &
echo \$! >/tmp/nwct.pid
sleep 2

echo \"pid=\$(cat /tmp/nwct.pid)\"
echo \"---- meminfo ----\"
free 2>/dev/null || true
cat /proc/meminfo 2>/dev/null | head -n 20 || true

echo \"---- listening ----\"
netstat -lnt 2>/dev/null || true

echo \"---- healthcheck: GET http://127.0.0.1:${NWCT_HTTP_PORT}/api/v1/config/init/status ----\"
if command -v curl >/dev/null 2>&1; then
  curl -fsS http://127.0.0.1:${NWCT_HTTP_PORT}/api/v1/config/init/status || true
elif command -v wget >/dev/null 2>&1; then
  wget -qO- http://127.0.0.1:${NWCT_HTTP_PORT}/api/v1/config/init/status || true
else
  echo \"(板子缺少 curl/wget，跳过 HTTP 检查)\"
fi

echo
echo \"---- last logs (/tmp/nwct.out) ----\"
tail -n 120 /tmp/nwct.out 2>/dev/null || true
'"


