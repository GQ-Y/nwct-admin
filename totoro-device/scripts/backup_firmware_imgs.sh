#!/usr/bin/env bash
set -euo pipefail

# 备份开发板“固件分区镜像”（.img）到本机目录：
# - 从 /proc/mtd 读取分区表
# - 逐个分区通过 /dev/mtdblockN 流式导出到本机文件（可选 gzip 压缩）
#
# 说明：
# - 这类 .img 是“分区级镜像”，对应你 luckfox 目录里的 env.img / idblock.img / uboot.img / boot.img / oem.img / userdata.img / rootfs.img
# - 如果你想生成“单文件 update.img”，那是 Rockchip/Luckfox 的打包格式，需要官方打包工具链；本脚本产出的是最通用、可回刷的分区镜像集合。
#
# 用法：
#   TARGET_HOST=192.168.2.182 TARGET_USER=root TARGET_PASS=luckfox OUT_DIR=./backups/pro_$(date +%Y%m%d_%H%M%S) ./scripts/backup_firmware_imgs.sh
#
# 可选：
#   COMPRESS=none|gzip   (默认 gzip)
#   SKIP_ROOTFS=1        (默认 0；rootfs 最大，若网络慢可先跳过)

TARGET_HOST="${TARGET_HOST:-}"
TARGET_USER="${TARGET_USER:-root}"
TARGET_PORT="${TARGET_PORT:-22}"
TARGET_PASS="${TARGET_PASS:-}"
OUT_DIR="${OUT_DIR:-./backups/firmware_imgs_$(date +%Y%m%d_%H%M%S)}"
COMPRESS="${COMPRESS:-none}" # none|gzip
SKIP_ROOTFS="${SKIP_ROOTFS:-0}"

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
  "${SSHPASS_PREFIX[@]}" ssh -n -p "${TARGET_PORT}" \
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

echo "[1/4] 读取 /proc/mtd"
mtd="$(ssh_cmd "cat /proc/mtd" | tr -d '\r')"
if [[ -z "${mtd}" ]]; then
  echo "读取 /proc/mtd 失败" >&2
  exit 2
fi
echo "${mtd}" >"${OUT_DIR}/proc_mtd.txt"

echo "[2/4] 检查设备端工具"
ssh_cmd "command -v dd >/dev/null 2>&1 || { echo 'missing dd'; exit 3; }"
if [[ "${COMPRESS}" == "gzip" ]]; then
  ssh_cmd "command -v gzip >/dev/null 2>&1 || { echo 'missing gzip'; exit 3; }"
fi

echo "[3/4] 开始导出分区镜像到: ${OUT_DIR}"

lines="$(echo "${mtd}" | tail -n +2 | sed '/^[[:space:]]*$/d' || true)"
if [[ -z "${lines}" ]]; then
  echo "未解析到任何 MTD 分区" >&2
  exit 4
fi

dump_part() {
  local dev="$1"     # mtd0
  local size_hex="$2"
  local name="$3"    # env

  if [[ -z "${name}" ]]; then
    name="${dev}"
  fi
  if [[ "${SKIP_ROOTFS}" == "1" ]] && [[ "${name}" == "rootfs" ]]; then
    echo "skip ${dev}(${name})"
    return 0
  fi

  local mtdnum="${dev#mtd}"
  local out="${OUT_DIR}/${name}.img"
  local final="${out}"
  if [[ "${COMPRESS}" == "gzip" ]]; then
    final="${out}.gz"
  fi

  echo "dump ${dev}(${name}) size=${size_hex} -> ${final}"

  mkdir -p "$(dirname "${final}")"

  # 按 /proc/mtd 的 size 精确读取，避免读多/读少；并避免远端 shell 的算术/变量导致本地 set -u 报错
  # size_hex 形如 00040000
  local size_dec
  size_dec="$((16#${size_hex}))"
  local bs=65536
  local cnt="$((size_dec / bs))"
  local skip="$((cnt * bs))"
  local rem="$((size_dec - skip))"

  if [[ "${COMPRESS}" == "gzip" ]]; then
    ssh_cmd "sh -lc 'set -e; dd if=/dev/mtdblock${mtdnum} bs=${bs} count=${cnt} 2>/dev/null; if [ ${rem} -gt 0 ]; then dd if=/dev/mtdblock${mtdnum} bs=1 skip=${skip} count=${rem} 2>/dev/null; fi' | gzip -1 -c" >"${final}"
  else
    ssh_cmd "sh -lc 'set -e; dd if=/dev/mtdblock${mtdnum} bs=${bs} count=${cnt} 2>/dev/null; if [ ${rem} -gt 0 ]; then dd if=/dev/mtdblock${mtdnum} bs=1 skip=${skip} count=${rem} 2>/dev/null; fi'" >"${final}"
  fi

  # 基本校验：文件不能太小（避免再次出现空文件/错误输出）
  local sz
  sz="$(wc -c <"${final}" 2>/dev/null | tr -d ' ' || true)"
  if [[ "${COMPRESS}" == "none" ]]; then
    expect="$((16#${size_hex}))"
    if [[ -z "${sz}" ]] || [[ "${sz}" != "${expect}" ]]; then
      echo "ERROR: 镜像大小不匹配：got=${sz:-} expect=${expect} file=${final}" >&2
      exit 10
    fi
  else
    if [[ "${sz}" -lt 1024 ]]; then
      echo "ERROR: 输出太小（${sz} bytes），疑似导出失败：${final}" >&2
      exit 10
    fi
  fi
}

while IFS= read -r ln; do
  dev="${ln%%:*}"
  rest="${ln#*:}"
  rest="$(echo "${rest}" | tr -s ' ')"
  size_hex="$(echo "${rest}" | cut -d' ' -f2)"
  name=""
  if [[ "${ln}" == *\"*\"* ]]; then
    name="${ln#*\"}"
    name="${name%%\"*}"
  fi
  dump_part "${dev}" "${size_hex}" "${name}"
done <<< "${lines}"

echo "[4/4] 完成：${OUT_DIR}"


