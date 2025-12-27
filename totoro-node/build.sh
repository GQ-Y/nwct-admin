#!/usr/bin/env bash
set -euo pipefail

# 用法：
#   ./build.sh                 # 本机构建
#   ./build.sh linux amd64     # 交叉编译
#   ./build.sh linux arm64
#
# 输出：
#   totoro-node/bin/totoro-node_<goos>_<goarch>

GOOS="${1:-$(go env GOOS)}"
GOARCH="${2:-$(go env GOARCH)}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${ROOT_DIR}/bin"
mkdir -p "${OUT_DIR}"

OUT="${OUT_DIR}/totoro-node_${GOOS}_${GOARCH}"

echo "Building totoro-node -> ${OUT}"
pushd "${ROOT_DIR}" >/dev/null
  env \
    CGO_ENABLED=0 \
    GOOS="${GOOS}" \
    GOARCH="${GOARCH}" \
    go build -trimpath -ldflags "-s -w" -o "${OUT}" ./cmd/totoro-node
popd >/dev/null

echo "Done."


