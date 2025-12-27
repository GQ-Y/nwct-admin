#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRAND_DIR="$ROOT_DIR/assets/branding"
OUT_DIR="$ROOT_DIR/build/branding_out"

APPICON_SET="$ROOT_DIR/macos/Runner/Assets.xcassets/AppIcon.appiconset"
WIN_ICON="$ROOT_DIR/windows/runner/resources/app_icon.ico"

mkdir -p "$OUT_DIR"

echo "[1/4] Render base app icon (1024) from SVG"
rsvg-convert -w 1024 -h 1024 "$BRAND_DIR/totoro_app_icon.svg" -o "$OUT_DIR/app_icon_1024.png"

echo "[2/4] Resize app icon PNGs for macOS AppIcon.appiconset"
for s in 16 32 64 128 256 512; do
  sips -z "$s" "$s" "$OUT_DIR/app_icon_1024.png" --out "$OUT_DIR/app_icon_${s}.png" >/dev/null
done

echo "[3/4] Copy into macOS xcassets"
cp -f "$OUT_DIR/app_icon_16.png" "$APPICON_SET/app_icon_16.png"
cp -f "$OUT_DIR/app_icon_32.png" "$APPICON_SET/app_icon_32.png"
cp -f "$OUT_DIR/app_icon_64.png" "$APPICON_SET/app_icon_64.png"
cp -f "$OUT_DIR/app_icon_128.png" "$APPICON_SET/app_icon_128.png"
cp -f "$OUT_DIR/app_icon_256.png" "$APPICON_SET/app_icon_256.png"
cp -f "$OUT_DIR/app_icon_512.png" "$APPICON_SET/app_icon_512.png"
cp -f "$OUT_DIR/app_icon_1024.png" "$APPICON_SET/app_icon_1024.png"

echo "[4/4] Build Windows .ico (PNG-embedded)"
for s in 16 24 32 48 64 128 256; do
  sips -z "$s" "$s" "$OUT_DIR/app_icon_1024.png" --out "$OUT_DIR/win_${s}.png" >/dev/null
done
/usr/bin/python3 "$ROOT_DIR/tools/make_ico.py" \
  "$WIN_ICON" \
  "$OUT_DIR/win_16.png" \
  "$OUT_DIR/win_24.png" \
  "$OUT_DIR/win_32.png" \
  "$OUT_DIR/win_48.png" \
  "$OUT_DIR/win_64.png" \
  "$OUT_DIR/win_128.png" \
  "$OUT_DIR/win_256.png"

echo "Done. macOS AppIcon + Windows .ico updated."


