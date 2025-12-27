#!/usr/bin/env bash
set -euo pipefail

# 固件备份（从设备拉到本机）：
# - 自动读取 /proc/mtd，逐个分区通过 nanddump 流式导出到本机文件
# - 支持 sshpass（密码登录）与 gzip 压缩
#
# 用法示例（Pro）：
#   TARGET_HOST=192.168.2.182 TARGET_USER=root TARGET_PASS=luckfox OUT_DIR=./backups/pro_$(date +%Y%m%d_%H%M%S) ./scripts/backup_firmware.sh
#
# 可选：
#   COMPRESS=gzip   # gzip | none（默认 gzip）
#   PARALLEL=0      # 0/1 是否并行导出（默认 0，避免压垮设备）

TARGET_HOST="${TARGET_HOST:-}"
TARGET_USER="${TARGET_USER:-root}"
TARGET_PORT="${TARGET_PORT:-22}"
TARGET_PASS="${TARGET_PASS:-}"
OUT_DIR="${OUT_DIR:-./backups/firmware_$(date +%Y%m%d_%H%M%S)}"
COMPRESS="${COMPRESS:-gzip}" # gzip|none
PARALLEL="${PARALLEL:-0}"

if [[ -z "${TARGET_HOST}" ]]; then
  echo "请提供 TARGET_HOST，例如：TARGET_HOST=192.168.2.182" >&2
  exit 1
fi

SSHPASS_PREFIX=()
if [[ -n "${TARGET_PASS}" ]]; then
  if ! command -v sshpass >/dev/null 2>&1; then
    echo "已设置 TARGET_PASS 但本机缺少 sshpass；请先安装：brew install sshpass" >&2
    exit 1
  fi
  SSHPASS_PREFIX=(sshpass -p "${TARGET_PASS}")
fi

ssh_cmd() {
  "${SSHPASS_PREFIX[@]}" ssh -p "${TARGET_PORT}" \
    -o ConnectTimeout=12 \
    -o StrictHostKeyChecking=accept-new \
    -o PubkeyAuthentication=no \
    -o PasswordAuthentication=yes \
    -o KbdInteractiveAuthentication=yes \
    -o PreferredAuthentications=password,keyboard-interactive \
    -o NumberOfPasswordPrompts=1 \
    "${TARGET_USER}@${TARGET_HOST}" "$@"
}

mkdir -p "${OUT_DIR}"

echo "[1/3] 读取分区表: ${TARGET_USER}@${TARGET_HOST}:/proc/mtd"
mtd="$(ssh_cmd "cat /proc/mtd" | tr -d '\r')"
if [[ -z "${mtd}" ]]; then
  echo "读取 /proc/mtd 失败（设备可能不是 NAND/MTD 方案）" >&2
  exit 2
fi
echo "${mtd}" | tee "${OUT_DIR}/proc_mtd.txt" >/dev/null

echo "[2/3] 检查设备端工具..."
ssh_cmd "command -v nanddump >/dev/null 2>&1 || { echo '设备缺少 nanddump'; exit 3; }"
if [[ "${COMPRESS}" == "gzip" ]]; then
  ssh_cmd "command -v gzip >/dev/null 2>&1 || { echo '设备缺少 gzip'; exit 3; }"
fi

echo "[3/3] 开始导出分区到本机: ${OUT_DIR}"

dump_one() {
  local mtddev="$1"   # mtd0
  local hexsz="$2"    # 00040000
  local name="$3"     # env

  local outfile="${OUT_DIR}/${mtddev}_${name}.bin"
  local final="${outfile}"

  if [[ "${COMPRESS}" == "gzip" ]]; then
    final="${outfile}.gz"
    echo "dump ${mtddev}(${name}) -> ${final}"
    ssh_cmd "sh -lc 'nanddump -f - /dev/${mtddev} 2>/dev/null | gzip -1 -c'" >"${final}"
  else
    echo "dump ${mtddev}(${name}) -> ${final}"
    ssh_cmd "sh -lc 'nanddump -f - /dev/${mtddev} 2>/dev/null'" >"${final}"
  fi
}

export -f dump_one
export OUT_DIR COMPRESS TARGET_HOST TARGET_USER TARGET_PORT TARGET_PASS

# 解析 /proc/mtd：
# mtd0: 00040000 00020000 "env"
lines="$(echo "${mtd}" | tail -n +2 | sed '/^\s*$/d' || true)"
if [[ -z "${lines}" ]]; then
  echo "未解析到任何 MTD 分区" >&2
  exit 4
fi

if [[ "${PARALLEL}" == "1" ]]; then
  # 简单并行：最多 2 路（避免把设备/网络打满）
  echo "${lines}" | while read -r ln; do
    dev="$(echo "${ln}" | cut -d: -f1)"
    rest="$(echo "${ln}" | cut -d: -f2-)"
    size_hex="$(echo "${rest}" | tr -s ' ' | cut -d' ' -f2)"
    name="$(echo "${ln}" | sed -n 's/.*\"\\(.*\\)\".*/\\1/p')"
    dump_one "${dev}" "${size_hex}" "${name}" &
    # 控制并发=2
    while [[ "$(jobs -pr | wc -l | tr -d ' ')" -ge 2 ]]; do
      sleep 0.2
    done
  done
  wait
else
  echo "${lines}" | while read -r ln; do
    dev="$(echo "${ln}" | cut -d: -f1)"
    rest="$(echo "${ln}" | cut -d: -f2-)"
    size_hex="$(echo "${rest}" | tr -s ' ' | cut -d' ' -f2)"
    name="$(echo "${ln}" | sed -n 's/.*\"\\(.*\\)\".*/\\1/p')"
    dump_one "${dev}" "${size_hex}" "${name}"
  done
fi

echo "完成：${OUT_DIR}"


