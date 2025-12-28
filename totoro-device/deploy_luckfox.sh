#!/usr/bin/env bash
set -euo pipefail

# 交互模式开关：
# - INTERACTIVE=1：脚本会提示输入 IP/账号/密码等（未提供时才会问）
# - INTERACTIVE=0：非交互（默认），仅使用环境变量/脚本默认值
INTERACTIVE="${INTERACTIVE:-0}"
if [[ "${INTERACTIVE}" == "1" ]] && [[ ! -t 0 ]]; then
  echo "检测到非交互终端（stdin 不是 TTY），已自动关闭 INTERACTIVE 模式。" >&2
  INTERACTIVE=0
fi

prompt_if_empty() {
  # prompt_if_empty VAR "提示" "默认值" [secret:0|1]
  local var_name="${1:?}"
  local prompt="${2:?}"
  local def="${3:-}"
  local secret="${4:-0}"

  # 通过间接引用取值；在 set -u 下需用 :- 防止未定义报错
  local cur=""
  cur="$(eval "printf '%s' \"\${${var_name}:-}\"")"
  if [[ -n "${cur}" ]]; then
    return 0
  fi

  if [[ "${INTERACTIVE}" != "1" ]]; then
    if [[ -n "${def}" ]]; then
      printf -v "${var_name}" '%s' "${def}"
    fi
    return 0
  fi

  local val=""
  if [[ "${secret}" == "1" ]]; then
    read -r -s -p "${prompt}" val
    echo
  else
    read -r -p "${prompt}" val
  fi

  if [[ -z "${val}" ]]; then
    val="${def}"
  fi
  printf -v "${var_name}" '%s' "${val}"
}

# Luckfox Pico-Ultra 默认参数（可通过环境变量覆盖）
TARGET_HOST="${TARGET_HOST:-192.168.2.221}"
TARGET_USER="${TARGET_USER:-root}"
TARGET_PORT="${TARGET_PORT:-22}"
# 先保留用户输入（用于判断是否“显式指定”）
TARGET_PATH_INPUT="${TARGET_PATH:-}"
TARGET_PASS_INPUT="${TARGET_PASS:-}"

TARGET_PATH="${TARGET_PATH_INPUT:-/root/totoro-device}"
TARGET_PASS="${TARGET_PASS_INPUT:-}"
# 设备类型：ultra=带屏；plus/pro=无屏（可按需扩展）
DEVICE_MODEL="${DEVICE_MODEL:-ultra}"
# 默认设备名称（编译期注入到二进制；也会写入测试 config.json）
DEVICE_NAME="${DEVICE_NAME:-}"
# 设备上测试端口：避免占用固件自带的 80 端口服务导致异常/重启
NWCT_HTTP_PORT="${NWCT_HTTP_PORT:-18080}"

# 交互式补全关键参数（仅在未提供时提示）
prompt_if_empty TARGET_HOST "请输入设备 IP（TARGET_HOST）[默认: ${TARGET_HOST}]: " "${TARGET_HOST}"
prompt_if_empty TARGET_USER "请输入登录账号（TARGET_USER）[默认: ${TARGET_USER}]: " "${TARGET_USER}"
prompt_if_empty TARGET_PORT "请输入 SSH 端口（TARGET_PORT）[默认: ${TARGET_PORT}]: " "${TARGET_PORT}"
prompt_if_empty TARGET_PASS "请输入 SSH 密码（TARGET_PASS，留空表示不使用密码/依赖 key）: " "" 1
prompt_if_empty DEVICE_MODEL "请输入设备型号（DEVICE_MODEL：ultra/plus/pro）[默认: ${DEVICE_MODEL}]: " "${DEVICE_MODEL}"
prompt_if_empty NWCT_HTTP_PORT "请输入设备端测试端口（NWCT_HTTP_PORT）[默认: ${NWCT_HTTP_PORT}]: " "${NWCT_HTTP_PORT}"
prompt_if_empty TARGET_PATH "请输入上传路径（TARGET_PATH）[默认: ${TARGET_PATH}]: " "${TARGET_PATH}"

# 编译 tags：ultra 默认带屏；plus/pro 默认无屏
BUILD_TAGS="${BUILD_TAGS:-}"
if [[ -z "${BUILD_TAGS}" ]]; then
  case "${DEVICE_MODEL}" in
    ultra|ultra-w|ultra_b|ultra_w)
      BUILD_TAGS="device_display"
      ;;
    plus|pro)
      BUILD_TAGS=""
      ;;
    *)
      # 未知型号：默认无屏，避免编译进 UI
      BUILD_TAGS=""
      ;;
  esac
fi

# 运行参数：
# - headless 版本不支持 -display 参数（编译期剔除 UI）
# - display 版本默认 -display=true
if [[ -z "${RUN_ARGS:-}" ]]; then
  if [[ -n "${BUILD_TAGS}" ]]; then
    RUN_ARGS="-display=true"
  else
    RUN_ARGS=""
  fi
fi

# 根据 DEVICE_MODEL 生成默认 DEVICE_NAME（可通过环境变量覆盖）
if [[ -z "${DEVICE_NAME}" ]]; then
  case "${DEVICE_MODEL}" in
    ultra|ultra-w|ultra_b|ultra_w)
      DEVICE_NAME="Totoro S1 Ultra"
      ;;
    plus)
      DEVICE_NAME="Totoro S1 Plus"
      ;;
    pro)
      DEVICE_NAME="Totoro S1 Pro"
      ;;
    *)
      DEVICE_NAME="Totoro S1"
      ;;
  esac
fi
if [[ "${DEVICE_NAME}" == *"'"* ]]; then
  echo "DEVICE_NAME 不能包含单引号（'）：${DEVICE_NAME}" >&2
  exit 1
fi

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
  if [[ ${#SSHPASS_PREFIX[@]} -gt 0 ]]; then
    "${SSHPASS_PREFIX[@]}" ssh -p "${TARGET_PORT}" \
      -o ConnectTimeout=8 \
      -o StrictHostKeyChecking=accept-new \
      -o PubkeyAuthentication=no \
      -o PasswordAuthentication=yes \
      -o KbdInteractiveAuthentication=yes \
      -o PreferredAuthentications=password,keyboard-interactive \
      -o NumberOfPasswordPrompts=1 \
      "${TARGET_USER}@${TARGET_HOST}" "$@"
  else
    ssh -p "${TARGET_PORT}" \
      -o ConnectTimeout=8 \
      -o StrictHostKeyChecking=accept-new \
      -o PubkeyAuthentication=no \
      -o PasswordAuthentication=yes \
      -o KbdInteractiveAuthentication=yes \
      -o PreferredAuthentications=password,keyboard-interactive \
      -o NumberOfPasswordPrompts=1 \
      "${TARGET_USER}@${TARGET_HOST}" "$@"
  fi
}

scp_cmd() {
  if [[ ${#SSHPASS_PREFIX[@]} -gt 0 ]]; then
    "${SSHPASS_PREFIX[@]}" scp -P "${TARGET_PORT}" \
      -o ConnectTimeout=8 \
      -o StrictHostKeyChecking=accept-new \
      -o PubkeyAuthentication=no \
      -o PasswordAuthentication=yes \
      -o KbdInteractiveAuthentication=yes \
      -o PreferredAuthentications=password,keyboard-interactive \
      -o NumberOfPasswordPrompts=1 \
      "$@"
  else
    scp -P "${TARGET_PORT}" \
      -o ConnectTimeout=8 \
      -o StrictHostKeyChecking=accept-new \
      -o PubkeyAuthentication=no \
      -o PasswordAuthentication=yes \
      -o KbdInteractiveAuthentication=yes \
      -o PreferredAuthentications=password,keyboard-interactive \
      -o NumberOfPasswordPrompts=1 \
      "$@"
  fi
}

echo "[1/4] 检测开发板架构..."
ARCH_RAW="$(ssh_cmd uname -m | tr -d '\r' | tr '[:upper:]' '[:lower:]' || true)"
if [[ -z "${ARCH_RAW}" ]]; then
  echo "无法通过 SSH 获取 uname -m；请确认能 ssh ${TARGET_USER}@${TARGET_HOST} 登录。" >&2
  exit 1
fi
echo "板子 uname -m: ${ARCH_RAW}"
RESOLVED_TARGET_PATH="${TARGET_PATH}"

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
OUT="${PROJECT_DIR}/bin/totoro-device_${GOOS}_${GOARCH}${GOARM:+v${GOARM}}${BUILD_TAGS:+_display}"
mkdir -p "${PROJECT_DIR}/bin"

pushd "${PROJECT_DIR}" >/dev/null
  env \
    CGO_ENABLED=0 \
    GOOS="${GOOS}" \
    GOARCH="${GOARCH}" \
    ${GOARM:+GOARM="${GOARM}"} \
    go build ${BUILD_TAGS:+-tags "${BUILD_TAGS}"} -trimpath -ldflags "-s -w -X 'totoro-device/config.DefaultDeviceName=${DEVICE_NAME}'" -o "${OUT}" .
popd >/dev/null

echo "编译产物: ${OUT}"

# 智能选择“上传路径”：必须满足可写 + 可用空间 >= 二进制体积（+余量）
if [[ -z "${TARGET_PATH_INPUT}" ]]; then
  BIN_SIZE="$(wc -c <"${OUT}" | tr -d ' ')"
  if [[ -z "${BIN_SIZE}" ]]; then
    BIN_SIZE=0
  fi
  # 10% 余量 + 至少 2MB
  NEED_BYTES="$((BIN_SIZE + BIN_SIZE/10 + 2*1024*1024))"
  NEED_KB="$(((NEED_BYTES + 1023) / 1024))"

  chosen_dir="$(ssh_cmd "sh -lc '
need_kb=${NEED_KB}

# 优先级：/mnt/sdcard（若有 TF 卡会更大；没挂载也会落到 rootfs 的 df）> /root > /userdata > /oem
for d in /mnt/sdcard /root /userdata /oem; do
  [ -d \"\$d\" ] || continue
  [ -w \"\$d\" ] || continue
  # df 第2行第4列=Avail(KB)，避免 awk 的字段引用
  avail_kb=\$(df -k \"\$d\" 2>/dev/null | sed -n \"2p\" | tr -s \" \" | cut -d\" \" -f4 | tr -d \" \")
  [ -n \"\$avail_kb\" ] || continue
  [ \"\$avail_kb\" -ge \"\$need_kb\" ] || continue
  echo \"\$d\"
  exit 0
done
echo /root
'")"
  chosen_dir="$(echo "${chosen_dir}" | tr -d '\r' | tail -n 1)"
  if [[ -n "${chosen_dir}" ]]; then
    RESOLVED_TARGET_PATH="${chosen_dir%/}/totoro-device"
    echo "已智能选择上传路径: ${RESOLVED_TARGET_PATH}（need_kb=${NEED_KB}）"
  fi
fi

echo "[3/4] 上传到开发板: ${TARGET_USER}@${TARGET_HOST}:${RESOLVED_TARGET_PATH}"
# 直接覆盖：删除旧的，不保留备份
# 同时尽量停掉旧进程、清理 staging，避免占满 /tmp（尤其是 Pro/Plus 的 tmpfs 较小）
ssh_cmd "sh -lc '
set -e
# 如果已安装自启脚本，先 stop（避免旧进程占用端口/资源）
if [ -x /etc/init.d/S99totoro-device ]; then
  /etc/init.d/S99totoro-device stop >/dev/null 2>&1 || true
fi
if [ -f /tmp/nwct.pid ]; then
  oldpid=\$(cat /tmp/nwct.pid 2>/dev/null || true)
  if [ -n \"\$oldpid\" ] && kill -0 \"\$oldpid\" 2>/dev/null; then
    kill \"\$oldpid\" 2>/dev/null || true
    sleep 1
  fi
fi
rm -rf /tmp/nwct/bin 2>/dev/null || true
rm -f /root/totoro-device 2>/dev/null || true
mkdir -p \"$(dirname "${RESOLVED_TARGET_PATH}")\"
rm -f '${RESOLVED_TARGET_PATH}'
'"
scp_cmd "${OUT}" "${TARGET_USER}@${TARGET_HOST}:${RESOLVED_TARGET_PATH}"
ssh_cmd "chmod +x '${RESOLVED_TARGET_PATH}'"

# 安装开机自启动脚本（部署时一并写入），保证“重启设备/恢复出厂”后服务会自动起来
INSTALL_AUTOSTART="${INSTALL_AUTOSTART:-1}"
if [[ "${INSTALL_AUTOSTART}" == "1" ]]; then
  echo "安装开机自启脚本: /etc/init.d/S99totoro-device"
  ssh_cmd "sh -lc '
set -e
mkdir -p /etc/init.d /var/run /var/log/nwct /var/nwct /etc/nwct
cat >/etc/init.d/S99totoro-device <<\"EOF\"
#!/bin/sh

### BEGIN INIT INFO
# Provides:          totoro-device
# Required-Start:    \$local_fs \$network
# Required-Stop:     \$local_fs \$network
# Default-Start:     S
# Default-Stop:      0 6
# Short-Description: Totoro Device service
### END INIT INFO

NAME=totoro-device
DAEMON=\"${RESOLVED_TARGET_PATH}\"
PIDFILE=/var/run/\${NAME}.pid
LOGFILE=/var/log/\${NAME}.log

export NWCT_CONFIG_PATH=/etc/nwct/config.json
export NWCT_DB_PATH=/var/nwct/devices.db
export NWCT_LOG_DIR=/var/log/nwct

# 可选品牌资源
export NWCT_BRANDING_PATH=\${NWCT_BRANDING_PATH:-/etc/nwct/branding.png}
export NWCT_BOOT_AUDIO_PATH=\${NWCT_BOOT_AUDIO_PATH:-/etc/nwct/boot.mp3}

start() {
  echo \"Starting \${NAME}...\"
  mkdir -p /var/run /var/log/nwct /var/nwct /etc/nwct

  if [ ! -x \"\${DAEMON}\" ]; then
    echo \"Missing executable: \${DAEMON}\"
    return 1
  fi

  if [ -f \"\${PIDFILE}\" ]; then
    pid=\$(cat \"\${PIDFILE}\" 2>/dev/null)
    if [ -n \"\${pid}\" ] && kill -0 \"\${pid}\" 2>/dev/null; then
      echo \"\${NAME} already running (pid=\${pid})\"
      return 0
    fi
  fi

  nohup \"\${DAEMON}\" ${RUN_ARGS} >>\"\${LOGFILE}\" 2>&1 &
  echo \$! >\"\${PIDFILE}\"
  sleep 1
  pid=\$(cat \"\${PIDFILE}\" 2>/dev/null)
  if [ -n \"\${pid}\" ] && kill -0 \"\${pid}\" 2>/dev/null; then
    echo \"\${NAME} started (pid=\${pid})\"
    return 0
  fi
  echo \"\${NAME} failed to start, see \${LOGFILE}\"
  return 1
}

stop() {
  echo \"Stopping \${NAME}...\"
  if [ ! -f \"\${PIDFILE}\" ]; then
    echo \"\${NAME} not running (no pidfile)\"
    return 0
  fi
  pid=\$(cat \"\${PIDFILE}\" 2>/dev/null)
  if [ -z \"\${pid}\" ]; then
    rm -f \"\${PIDFILE}\"
    return 0
  fi
  kill \"\${pid}\" 2>/dev/null || true
  for i in 1 2 3 4 5 6 7 8 9 10; do
    if kill -0 \"\${pid}\" 2>/dev/null; then
      sleep 1
    else
      break
    fi
  done
  if kill -0 \"\${pid}\" 2>/dev/null; then
    kill -9 \"\${pid}\" 2>/dev/null || true
  fi
  rm -f \"\${PIDFILE}\"
  echo \"\${NAME} stopped\"
}

status() {
  if [ -f \"\${PIDFILE}\" ]; then
    pid=\$(cat \"\${PIDFILE}\" 2>/dev/null)
    if [ -n \"\${pid}\" ] && kill -0 \"\${pid}\" 2>/dev/null; then
      echo \"\${NAME} running (pid=\${pid})\"
      return 0
    fi
  fi
  echo \"\${NAME} not running\"
  return 3
}

case \"\$1\" in
  start) start ;;
  stop) stop ;;
  restart) stop; start ;;
  status) status ;;
  *) echo \"Usage: \$0 {start|stop|restart|status}\"; exit 1 ;;
esac
EOF
chmod +x /etc/init.d/S99totoro-device
'"
fi

echo "[4/4] 远程运行测试（前台运行，Ctrl+C 可中断；或你也可以改用 nohup 后台）..."
DAEMON_SRC="${RESOLVED_TARGET_PATH}"
DAEMON_RUN="${RESOLVED_TARGET_PATH}"
echo "执行（后台）：${DAEMON_RUN} ${RUN_ARGS}"
ssh_cmd "sh -lc '
set -e

# 使用 /tmp 下的“测试配置”，避免改动系统 /etc 配置、避免占用 80 端口
mkdir -p /tmp/nwct /tmp/nwct/log /tmp/nwct/bin

# 智能选择 NWCT_CACHE_DIR（给 frpc 落盘用）：必须可写 + 可执行 + 空间足够
# - frpc 大约 15MB，这里按 20MB 预留（含余量）
if [ -z \"${NWCT_CACHE_DIR:-}\" ]; then
  need_kb=20480
  cache_dir=\"\"
  for d in /root/.cache/nwct/bin /userdata/.cache/nwct/bin /oem/.cache/nwct/bin /tmp/nwct/cache/bin /mnt/sdcard/.cache/nwct/bin; do
    [ -d \"\$d\" ] || mkdir -p \"\$d\" 2>/dev/null || continue
    [ -w \"\$d\" ] || continue
    avail_kb=\$(df -k \"\$d\" 2>/dev/null | sed -n \"2p\" | tr -s \" \" | cut -d\" \" -f4 | tr -d \" \")
    [ -n \"\$avail_kb\" ] || continue
    [ \"\$avail_kb\" -ge \"\$need_kb\" ] || continue
    # 可执行探测（识别 vfat/noexec）
    t=\"\$d/.totoro_exec_test.sh\"
    echo \"#!/bin/sh\" >\"\$t\" 2>/dev/null || continue
    echo \"exit 0\" >>\"\$t\" 2>/dev/null || { rm -f \"\$t\" 2>/dev/null || true; continue; }
    chmod +x \"\$t\" 2>/dev/null || { rm -f \"\$t\" 2>/dev/null || true; continue; }
    \"\$t\" >/dev/null 2>&1 || { rm -f \"\$t\" 2>/dev/null || true; continue; }
    rm -f \"\$t\" 2>/dev/null || true
    cache_dir=\"\$d\"
    break
  done
  if [ -n \"\$cache_dir\" ]; then
    export NWCT_CACHE_DIR=\"\$cache_dir\"
  fi
fi

# 判断源目录是否支持执行（某些 TF 卡 vfat 会因 fmask/noexec 导致不可执行）
DAEMON_SRC=\"${DAEMON_SRC}\"
DAEMON_RUN=\"${DAEMON_RUN}\"
BIN_DIR=\$(dirname \"\$DAEMON_SRC\")
EXEC_OK=0
if [ -d \"\$BIN_DIR\" ] && [ -w \"\$BIN_DIR\" ]; then
  TEST_SH=\"\$BIN_DIR/.totoro_exec_test.sh\"
  echo \"#!/bin/sh\" >\"\$TEST_SH\" 2>/dev/null || true
  echo \"exit 0\" >>\"\$TEST_SH\" 2>/dev/null || true
  chmod +x \"\$TEST_SH\" 2>/dev/null || true
  \"\$TEST_SH\" >/dev/null 2>&1 && EXEC_OK=1 || EXEC_OK=0
  rm -f \"\$TEST_SH\" >/dev/null 2>&1 || true
fi

if [ \"\$EXEC_OK\" != \"1\" ]; then
  # 需要 staging：优先 /tmp（若空间足够），否则回落到 /root（通常空间更大）
  # 计算二进制大小（字节）与 /tmp 可用空间（字节）
  # 避免在本地 bash(set -u) 中触发位置参数展开：不使用 awk 的字段引用（例如“第1列/第2列”这种）
  BIN_SIZE=\$(wc -c <\"\$DAEMON_SRC\" 2>/dev/null | tr -d \" \")
  TMP_AVAIL_KB=\$(df -k /tmp 2>/dev/null | sed -n \"2p\" | tr -s \" \" | cut -d\" \" -f4 | tr -d \" \")
  TMP_AVAIL_BYTES=\$((TMP_AVAIL_KB * 1024))
  # 给 10% 余量
  NEED_BYTES=\$((BIN_SIZE + BIN_SIZE/10))
  if [ -n \"\$BIN_SIZE\" ] && [ \"\$TMP_AVAIL_BYTES\" -gt \"\$NEED_BYTES\" ]; then
    DAEMON_RUN=\"/tmp/nwct/bin/totoro-device\"
  else
    DAEMON_RUN=\"/root/totoro-device\"
  fi
fi

# 若需要 staging，则复制到可执行位置再运行
if [ \"\$DAEMON_SRC\" != \"\$DAEMON_RUN\" ]; then
  cp -f \"\$DAEMON_SRC\" \"\$DAEMON_RUN\"
  chmod +x \"\$DAEMON_RUN\"
fi
cat >/tmp/nwct/config.json <<EOF
{
  \"initialized\": false,
  \"device\": {\"id\": \"DEV001\", \"name\": \"${DEVICE_NAME}\"},
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

nohup \"${DAEMON_RUN}\" ${RUN_ARGS} >/tmp/nwct.out 2>&1 &
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


