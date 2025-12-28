#!/bin/bash
# 将 Plus 的 MicroSD 镜像烧录到 SD 卡，并安装 totoro-device

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_DIR="${PROJECT_DIR}/../luckfox/镜像文件/buildroot（推荐）/MicroSD"
IMAGE_ZIP="${IMAGE_DIR}/Luckfox_Pico_Plus_MicroSD_250607.zip"

if [[ ! -f "${IMAGE_ZIP}" ]]; then
  echo "错误：未找到镜像文件 ${IMAGE_ZIP}" >&2
  exit 1
fi

# 检查是否有 SD 卡设备
echo "请插入 SD 卡到电脑..."
echo "正在检测 SD 卡设备..."
sleep 2

# macOS 检测
if [[ "$(uname)" == "Darwin" ]]; then
  echo "检测到的磁盘设备："
  diskutil list | grep -E "^/dev/disk[0-9]+" || true
  echo ""
  read -p "请输入 SD 卡设备路径（如 /dev/disk2，不要带分区号）：" SD_DEVICE
  
  if [[ ! -b "${SD_DEVICE}" ]]; then
    echo "错误：${SD_DEVICE} 不是块设备" >&2
    exit 1
  fi
  
  # 确认
  echo ""
  echo "警告：将格式化 ${SD_DEVICE}，所有数据将被清除！"
  read -p "确认继续？(yes/no): " CONFIRM
  if [[ "${CONFIRM}" != "yes" ]]; then
    echo "已取消"
    exit 0
  fi
  
  # 卸载所有分区
  echo "卸载 SD 卡分区..."
  for partition in $(diskutil list "${SD_DEVICE}" | grep -E "^/dev/disk[0-9]+s[0-9]+" | awk '{print $1}'); do
    diskutil unmount "${partition}" 2>/dev/null || true
  done
  
  # 解压镜像
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
      OFFSET_LINE=$(grep "^#${PART_NAME}.img" "${SD_UPDATE_FILE}")
      if [[ -n "${OFFSET_LINE}" ]]; then
        OFFSET_HEX=$(echo "${OFFSET_LINE}" | awk '{print $2}' | cut -d':' -f1)
        if [[ -n "${OFFSET_HEX}" ]]; then
          # 转换为字节（扇区大小 = 512 字节）
          OFFSET=$((OFFSET_HEX * 512))
          echo "烧录 ${PART_NAME}.img (大小: $(du -h "${PART_IMG}" | cut -f1)) 到偏移 ${OFFSET}..."
          sudo dd if="${PART_IMG}" of="${SD_DEVICE}" bs=512 seek=$((OFFSET / 512)) conv=fsync
          echo "✅ ${PART_NAME}.img 烧录完成"
        fi
      fi
    done
    
    sudo sync
    echo ""
    echo "✅ 所有分区烧录完成！"
  else
    # 回退到使用 update.img（完整镜像一键烧录）
    echo "警告：未找到 sd_update.txt，使用 update.img 完整镜像烧录..."
    IMG_FILE=$(find "${IMAGE_SUBDIR}" -name "update.img" | head -1)
    if [[ -z "${IMG_FILE}" ]]; then
      echo "错误：未找到 update.img 文件" >&2
      exit 1
    fi
    echo "找到镜像文件: ${IMG_FILE}"
    echo "镜像大小: $(du -h "${IMG_FILE}" | cut -f1)"
    echo "正在烧录镜像到 ${SD_DEVICE}（这可能需要几分钟）..."
    sudo dd if="${IMG_FILE}" of="${SD_DEVICE}" bs=4m status=progress conv=fsync
  fi
  
  echo ""
  echo "烧录完成！"
  echo "请将 SD 卡插入 Plus 开发板，然后："
  echo "1. 确保开发板从 SD 卡启动（可能需要设置启动顺序）"
  echo "2. 连接开发板到网络"
  echo "3. 使用以下命令部署 totoro-device："
  echo ""
  echo "   cd ${PROJECT_DIR}"
  echo "   BRIDGE_API_URL=\"http://192.168.2.32:18090\" \\"
  echo "   TARGET_HOST=<开发板IP> \\"
  echo "   TARGET_USER=root \\"
  echo "   TARGET_PASS=luckfox \\"
  echo "   DEVICE_MODEL=plus \\"
  echo "   INTERACTIVE=0 \\"
  echo "   ./deploy_luckfox.sh"
  
elif [[ "$(uname)" == "Linux" ]]; then
  echo "检测到的磁盘设备："
  lsblk -d -o NAME,SIZE,MODEL | grep -E "^[a-z]+" || true
  echo ""
  read -p "请输入 SD 卡设备路径（如 /dev/sdb，不要带分区号）：" SD_DEVICE
  
  if [[ ! -b "${SD_DEVICE}" ]]; then
    echo "错误：${SD_DEVICE} 不是块设备" >&2
    exit 1
  fi
  
  # 确认
  echo ""
  echo "警告：将格式化 ${SD_DEVICE}，所有数据将被清除！"
  read -p "确认继续？(yes/no): " CONFIRM
  if [[ "${CONFIRM}" != "yes" ]]; then
    echo "已取消"
    exit 0
  fi
  
  # 卸载所有分区
  echo "卸载 SD 卡分区..."
  for partition in $(lsblk -ln -o NAME "${SD_DEVICE}" | grep -v "^${SD_DEVICE##*/}$"); do
    umount "/dev/${partition}" 2>/dev/null || true
  done
  
  # 解压镜像
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
      OFFSET_LINE=$(grep "^#${PART_NAME}.img" "${SD_UPDATE_FILE}")
      if [[ -n "${OFFSET_LINE}" ]]; then
        OFFSET_HEX=$(echo "${OFFSET_LINE}" | awk '{print $2}' | cut -d':' -f1)
        if [[ -n "${OFFSET_HEX}" ]]; then
          # 转换为字节（扇区大小 = 512 字节）
          OFFSET=$((OFFSET_HEX * 512))
          echo "烧录 ${PART_NAME}.img (大小: $(du -h "${PART_IMG}" | cut -f1)) 到偏移 ${OFFSET}..."
          sudo dd if="${PART_IMG}" of="${SD_DEVICE}" bs=512 seek=$((OFFSET / 512)) conv=fsync
          echo "✅ ${PART_NAME}.img 烧录完成"
        fi
      fi
    done
    
    sudo sync
    echo ""
    echo "✅ 所有分区烧录完成！"
  else
    # 回退到使用 update.img（完整镜像一键烧录）
    echo "警告：未找到 sd_update.txt，使用 update.img 完整镜像烧录..."
    IMG_FILE=$(find "${IMAGE_SUBDIR}" -name "update.img" | head -1)
    if [[ -z "${IMG_FILE}" ]]; then
      echo "错误：未找到 update.img 文件" >&2
      exit 1
    fi
    echo "找到镜像文件: ${IMG_FILE}"
    echo "镜像大小: $(du -h "${IMG_FILE}" | cut -f1)"
    echo "正在烧录镜像到 ${SD_DEVICE}（这可能需要几分钟）..."
    sudo dd if="${IMG_FILE}" of="${SD_DEVICE}" bs=4M status=progress conv=fsync
  fi
  
  echo ""
  echo "烧录完成！"
  echo "请将 SD 卡插入 Plus 开发板，然后："
  echo "1. 确保开发板从 SD 卡启动（可能需要设置启动顺序）"
  echo "2. 连接开发板到网络"
  echo "3. 使用以下命令部署 totoro-device："
  echo ""
  echo "   cd ${PROJECT_DIR}"
  echo "   BRIDGE_API_URL=\"http://192.168.2.32:18090\" \\"
  echo "   TARGET_HOST=<开发板IP> \\"
  echo "   TARGET_USER=root \\"
  echo "   TARGET_PASS=luckfox \\"
  echo "   DEVICE_MODEL=plus \\"
  echo "   INTERACTIVE=0 \\"
  echo "   ./deploy_luckfox.sh"
else
  echo "错误：不支持的操作系统 $(uname)" >&2
  exit 1
fi

