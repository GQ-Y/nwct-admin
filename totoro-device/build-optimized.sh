#!/usr/bin/env bash
set -euo pipefail

# 为嵌入式设备（如 Lyra Plus）优化的编译脚本
# 目标：减小二进制体积，适配低内存/低存储环境

TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-arm}"
OUTPUT="${OUTPUT:-nwct-client}"

echo "=========================================="
echo "优化编译 nwct-client 为嵌入式设备"
echo "=========================================="
echo "目标平台: ${TARGET_OS}/${TARGET_ARCH}"
echo "输出文件: ${OUTPUT}"
echo ""

# 检查是否需要交叉编译
if [[ "${TARGET_OS}" != "$(go env GOOS)" ]] || [[ "${TARGET_ARCH}" != "$(go env GOARCH)" ]]; then
    echo "设置交叉编译环境变量..."
    export GOOS="${TARGET_OS}"
    export GOARCH="${TARGET_ARCH}"
    # 对于 ARM，可能需要指定 ARM 版本
    if [[ "${TARGET_ARCH}" == "arm" ]]; then
        # Cortex-A7 是 ARMv7，使用 GOARM=7
        export GOARM=7
        echo "  GOOS=${GOOS}"
        echo "  GOARCH=${GOARCH}"
        echo "  GOARM=${GOARM}"
    fi
fi

echo ""
echo "开始编译（使用优化选项）..."
echo ""

# 优化编译选项：
# -ldflags="-s -w": 去除符号表和调试信息，减小体积
# -trimpath: 去除文件系统路径信息
# -buildmode=exe: 标准可执行文件（默认，但明确指定）
go build \
    -trimpath \
    -ldflags="-s -w" \
    -buildmode=exe \
    -o "${OUTPUT}" \
    .

if [[ ! -f "${OUTPUT}" ]]; then
    echo "错误: 编译失败，未生成输出文件" >&2
    exit 1
fi

# 获取编译后的文件大小
SIZE_BEFORE=$(stat -f%z "${OUTPUT}" 2>/dev/null || stat -c%s "${OUTPUT}" 2>/dev/null || echo "0")
SIZE_BEFORE_MB=$(awk "BEGIN {printf \"%.2f\", ${SIZE_BEFORE}/1024/1024}")

echo ""
echo "编译完成！"
echo "文件大小: ${SIZE_BEFORE_MB} MB (${SIZE_BEFORE} 字节)"
echo ""

# 尝试 strip 进一步减小体积（如果可用）
if command -v strip >/dev/null 2>&1; then
    echo "执行 strip 去除符号表..."
    strip "${OUTPUT}" 2>/dev/null || true
    SIZE_AFTER=$(stat -f%z "${OUTPUT}" 2>/dev/null || stat -c%s "${OUTPUT}" 2>/dev/null || echo "0")
    SIZE_AFTER_MB=$(awk "BEGIN {printf \"%.2f\", ${SIZE_AFTER}/1024/1024}")
    REDUCTION=$(awk "BEGIN {printf \"%.2f\", (${SIZE_BEFORE} - ${SIZE_AFTER})/1024/1024}")
    echo "  strip 后: ${SIZE_AFTER_MB} MB (减少 ${REDUCTION} MB)"
    echo ""
fi

# 显示最终文件信息
echo "最终文件信息:"
file "${OUTPUT}" || true
ls -lh "${OUTPUT}" || true

echo ""
echo "=========================================="
echo "编译完成！"
echo "=========================================="
echo ""
echo "注意事项："
echo "1. 此二进制适用于 ${TARGET_OS}/${TARGET_ARCH} 平台"
if [[ "${TARGET_OS}" == "linux" ]] && [[ "${TARGET_ARCH}" == "arm" ]]; then
    echo "2. 适用于 ARM Cortex-A7 架构（如 Lyra Plus）"
fi
echo "3. 文件已优化，去除了调试信息和符号表"
echo "4. 如需进一步压缩，可考虑使用 UPX（但会增加启动时间）"
echo ""

