#!/bin/bash
# 在 Plus 开发板上直接烧录 MicroSD 镜像到 SD 卡
# 适用于开发板已插入 SD 卡的情况

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_DIR="${PROJECT_DIR}/../luckfox/镜像文件/buildroot（推荐）/MicroSD"
IMAGE_ZIP="${IMAGE_DIR}/Luckfox_Pico_Plus_MicroSD_250607.zip"

# 默认连接参数
TARGET_HOST="${TARGET_HOST:-192.168.2.226}"
TARGET_USER="${TARGET_USER:-root}"
TARGET_PASS="${TARGET_PASS:-luckfox}"
TARGET_PORT="${TARGET_PORT:-22}"
# 非交互式模式（设置 FORCE=1 跳过确认）
FORCE="${FORCE:-0}"

if [[ ! -f "${IMAGE_ZIP}" ]]; then
  echo "错误：未找到镜像文件 ${IMAGE_ZIP}" >&2
  exit 1
fi

# SSH 命令封装
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

echo "正在检查开发板上的 SD 卡..."
SD_DEVICE=$(ssh_cmd 'cat /proc/partitions | grep mmcblk | head -1 | awk "{print \$4}" | grep "^mmcblk[0-9]$" || echo ""')
if [[ -z "${SD_DEVICE}" ]]; then
  echo "错误：未在开发板上检测到 SD 卡设备" >&2
  echo "请确保 SD 卡已插入开发板" >&2
  exit 1
fi

echo "检测到 SD 卡设备: /dev/${SD_DEVICE}"

# 检查 SD 卡是否已挂载
MOUNTED=$(ssh_cmd "mount | grep ${SD_DEVICE} | head -1 || echo ''")
if [[ -n "${MOUNTED}" ]]; then
  echo "警告：SD 卡已挂载，正在卸载..."
  ssh_cmd "umount /dev/${SD_DEVICE}* 2>/dev/null || true"
  sleep 1
fi

# 确认
echo ""
echo "警告：将格式化 /dev/${SD_DEVICE}，所有数据将被清除！"
if [[ "${FORCE}" != "1" ]]; then
  read -p "确认继续？(yes/no): " CONFIRM
  if [[ "${CONFIRM}" != "yes" ]]; then
    echo "已取消"
    exit 0
  fi
else
  echo "非交互式模式：自动确认"
fi

# 解压镜像到临时目录
echo "解压镜像文件..."
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT
unzip -q "${IMAGE_ZIP}" -d "${TEMP_DIR}"

IMAGE_SUBDIR=$(find "${TEMP_DIR}" -type d -name "Luckfox_Pico_Plus_MicroSD_*" | head -1)
if [[ -z "${IMAGE_SUBDIR}" ]]; then
  echo "错误：未找到镜像子目录" >&2
  exit 1
fi

echo "找到镜像目录: ${IMAGE_SUBDIR}"

# 优先使用按分区烧录（更准确）- 解析 sd_update.txt 获取分区偏移
SD_UPDATE_FILE="${IMAGE_SUBDIR}/sd_update.txt"
if [[ -f "${SD_UPDATE_FILE}" ]]; then
  echo "使用按分区烧录方式（推荐）..."
    echo "解析分区布局（从 sd_update.txt）..."
    # 分区顺序：env, idblock, uboot, boot, oem, userdata, rootfs
    PARTITION_ORDER=("env" "idblock" "uboot" "boot" "oem" "userdata" "rootfs")
    
    for PART_NAME in "${PARTITION_ORDER[@]}"; do
      PART_IMG="${IMAGE_SUBDIR}/${PART_NAME}.img"
      if [[ ! -f "${PART_IMG}" ]]; then
        echo "警告：${PART_NAME}.img 不存在，跳过"
        continue
      fi
      
      # 从 sd_update.txt 获取偏移（格式：#env.img 0x0:0x0 0x40:0x8000 ...）
      # 第二个字段是起始扇区地址（十六进制）
      OFFSET_LINE=$(grep "^#${PART_NAME}.img" "${SD_UPDATE_FILE}")
      if [[ -n "${OFFSET_LINE}" ]]; then
        # 提取第二个字段（0x0:0x0 中的第一个 0x0）
        OFFSET_HEX=$(echo "${OFFSET_LINE}" | awk '{print $2}' | cut -d':' -f1)
        if [[ -n "${OFFSET_HEX}" ]]; then
          # 转换为字节（扇区大小 = 512 字节）
          OFFSET=$((OFFSET_HEX * 512))
          echo "烧录 ${PART_NAME}.img (大小: $(du -h "${PART_IMG}" | cut -f1)) 到偏移 ${OFFSET}..."
          
          # 流式传输并烧录
          if command -v pv >/dev/null 2>&1; then
            pv "${PART_IMG}" | ssh_cmd "dd of=/dev/${SD_DEVICE} bs=512 seek=$((OFFSET / 512)) conv=fsync 2>/dev/null"
          else
            cat "${PART_IMG}" | ssh_cmd "dd of=/dev/${SD_DEVICE} bs=512 seek=$((OFFSET / 512)) conv=fsync 2>/dev/null"
          fi
          echo "✅ ${PART_NAME}.img 烧录完成"
        else
          echo "警告：无法解析 ${PART_NAME} 的偏移信息，跳过"
        fi
      else
        echo "警告：未找到 ${PART_NAME} 的分区信息，跳过"
      fi
    done
    
  # 同步
  ssh_cmd "sync"
  echo ""
  echo "✅ 所有分区烧录完成！"
else
  # 回退到 update.img 完整镜像烧录
  echo "使用 update.img 完整镜像烧录..."
  IMG_FILE="${IMAGE_SUBDIR}/update.img"
  if [[ ! -f "${IMG_FILE}" ]]; then
    echo "错误：未找到 update.img 文件" >&2
    exit 1
  fi
  echo "镜像大小: $(du -h "${IMG_FILE}" | cut -f1)"
  
  if command -v pv >/dev/null 2>&1; then
    pv "${IMG_FILE}" | ssh_cmd "dd of=/dev/${SD_DEVICE} bs=4M conv=fsync && sync"
  else
    echo "正在烧录（这可能需要几分钟，请耐心等待）..."
    cat "${IMG_FILE}" | ssh_cmd "dd of=/dev/${SD_DEVICE} bs=4M conv=fsync && sync"
  fi
fi

echo ""
echo "烧录完成！"
echo ""
echo "重要提示："
echo "1. 开发板需要从 SD 卡启动才能使用 SD 卡上的系统"
echo "2. 如果开发板仍从内置存储启动（storagemedia=mtd），请尝试："
echo "   - 按住 BOOT 键的同时重启开发板"
echo "   - 或者使用 luckfox-config 配置启动顺序"
echo "   - 或者检查开发板是否有启动顺序设置"
echo ""
echo "3. 重启开发板后，检查是否从 SD 卡启动："
echo "   ssh root@<开发板IP> 'cat /proc/cmdline | grep storagemedia'"
echo "   如果显示 'storagemedia=mmc' 或 'root=/dev/mmcblk1p7'，说明已从 SD 卡启动"
echo ""
echo "4. 从 SD 卡启动后，使用以下命令部署 totoro-device："
echo ""
echo "   cd ${PROJECT_DIR}"
echo "   BRIDGE_API_URL=\"http://192.168.2.32:18090\" \\"
echo "   TARGET_HOST=<开发板IP> \\"
echo "   TARGET_USER=root \\"
echo "   TARGET_PASS=luckfox \\"
echo "   DEVICE_MODEL=plus \\"
echo "   INTERACTIVE=0 \\"
echo "   ./deploy_luckfox.sh"

