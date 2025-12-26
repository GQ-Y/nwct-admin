#!/usr/bin/env bash
set -euo pipefail

# nwct-client 控制脚本：start/stop/restart/status/logs
#
# 设计目标：
# - 适合开发板/嵌入式环境：无需 systemd 也能后台守护式运行
# - 使用 pidfile 管理进程，避免误杀其它同名进程
# - 允许通过环境变量覆盖关键路径

CMD="${1:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${NWCT_BIN:-${SCRIPT_DIR}/nwct-client}"

PIDFILE="${NWCT_PIDFILE:-${SCRIPT_DIR}/nwct-client.pid}"

detect_config_path() {
  if [[ -n "${NWCT_CONFIG_PATH:-}" ]]; then
    echo "${NWCT_CONFIG_PATH}"
    return
  fi
  if [[ -f "/etc/nwct/config.json" ]]; then
    echo "/etc/nwct/config.json"
    return
  fi
  # repo 模式：client-nps/../config.json
  if [[ -f "${SCRIPT_DIR}/../config.json" ]]; then
    echo "${SCRIPT_DIR}/../config.json"
    return
  fi
  # 兜底：当前目录
  echo "${SCRIPT_DIR}/config.json"
}

detect_log_dir() {
  if [[ -n "${NWCT_LOG_DIR:-}" ]]; then
    echo "${NWCT_LOG_DIR}"
    return
  fi
  if [[ -d "/var/log" && -w "/var/log" ]]; then
    echo "/var/log/nwct"
    return
  fi
  echo "/tmp/nwct"
}

CONFIG_PATH="$(detect_config_path)"
LOG_DIR="$(detect_log_dir)"
LOG_FILE="${NWCT_LOG_FILE:-${LOG_DIR}/system.log}"

mkdir -p "${LOG_DIR}"

is_running() {
  if [[ ! -f "${PIDFILE}" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "${PIDFILE}" 2>/dev/null || true)"
  [[ -n "${pid}" ]] || return 1
  kill -0 "${pid}" 2>/dev/null
}

print_status() {
  if is_running; then
    echo "running (pid=$(cat "${PIDFILE}"))"
  else
    echo "stopped"
  fi
}

start() {
  if [[ ! -x "${BIN}" ]]; then
    echo "ERROR: 找不到可执行文件或不可执行: ${BIN}" >&2
    echo "提示：请先在 ${SCRIPT_DIR} 下构建：go build -o nwct-client ." >&2
    exit 1
  fi

  if is_running; then
    echo "already running (pid=$(cat "${PIDFILE}"))"
    exit 0
  fi

  # 80 端口通常需要 root/capabilities；由调用者决定是否 sudo 执行脚本
  echo "starting..."
  echo "  BIN=${BIN}"
  echo "  CONFIG=${CONFIG_PATH}"
  echo "  LOG=${LOG_FILE}"

  # 后台启动：nohup + pidfile
  # shellcheck disable=SC2091
  nohup env \
    NWCT_CONFIG_PATH="${CONFIG_PATH}" \
    NWCT_LOG_DIR="${LOG_DIR}" \
    "${BIN}" \
    >>"${LOG_FILE}" 2>&1 &

  echo $! > "${PIDFILE}"
  sleep 0.2
  if is_running; then
    echo "started (pid=$(cat "${PIDFILE}"))"
  else
    echo "ERROR: start failed, see log: ${LOG_FILE}" >&2
    exit 1
  fi
}

stop() {
  if ! is_running; then
    echo "not running"
    rm -f "${PIDFILE}" || true
    exit 0
  fi

  local pid
  pid="$(cat "${PIDFILE}")"
  echo "stopping (pid=${pid})..."
  kill -TERM "${pid}" 2>/dev/null || true

  # 等待优雅退出
  for _ in {1..50}; do
    if ! kill -0 "${pid}" 2>/dev/null; then
      rm -f "${PIDFILE}" || true
      echo "stopped"
      return
    fi
    sleep 0.1
  done

  echo "still running, force kill (pid=${pid})..."
  kill -KILL "${pid}" 2>/dev/null || true
  rm -f "${PIDFILE}" || true
  echo "killed"
}

restart() {
  stop || true
  start
}

logs() {
  if [[ ! -f "${LOG_FILE}" ]]; then
    echo "log file not found: ${LOG_FILE}"
    exit 1
  fi
  tail -n 200 "${LOG_FILE}"
}

follow() {
  if [[ ! -f "${LOG_FILE}" ]]; then
    echo "log file not found: ${LOG_FILE}"
    exit 1
  fi
  tail -f "${LOG_FILE}"
}

usage() {
  cat <<EOF
Usage:
  $(basename "$0") start|stop|restart|status|logs|follow

Env overrides:
  NWCT_BIN          binary path (default: ./nwct-client)
  NWCT_CONFIG_PATH  config path (default: /etc/nwct/config.json or ../config.json)
  NWCT_LOG_DIR      log dir (default: /var/log/nwct or /tmp/nwct)
  NWCT_LOG_FILE     log file (default: \${NWCT_LOG_DIR}/system.log)
  NWCT_PIDFILE      pidfile path (default: ./nwct-client.pid)
EOF
}

case "${CMD}" in
  start)   start ;;
  stop)    stop ;;
  restart) restart ;;
  status)  print_status ;;
  logs)    logs ;;
  follow)  follow ;;
  ""|-h|--help|help) usage ;;
  *)
    echo "unknown command: ${CMD}" >&2
    usage
    exit 2
    ;;
esac


