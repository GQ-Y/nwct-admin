# NWCT 显示系统 - 纯 Go 混合驱动方案

## 概述

基于纯 Go 实现的屏幕交互系统，支持在 Mac M1 开发环境使用 SDL2 预览，生产环境直接操作 Framebuffer。

### 特性

- ✅ 纯 Go 实现（生产环境无 CGO 依赖）
- ✅ 混合驱动架构（开发/生产分离）
- ✅ 480x480 分辨率支持
- ✅ 鸿蒙风格炫彩 UI
- ✅ 30 FPS 流畅渲染
- ✅ 实时数据展示（网络速度、隧道数量、运行时间）

## 快速开始

### 1. 安装依赖

```bash
# Mac M1 开发环境
brew install sdl2

# 安装 Go 依赖
cd client-nps
go get github.com/veandco/go-sdl2/sdl
```

### 2. 构建和运行

```bash
# 使用 Makefile（推荐）
make -f Makefile.display run-preview

# 或手动构建
go build -tags preview -o bin/nwct-preview ./cmd/display-test/
./bin/nwct-preview
```

### 3. 查看效果

程序会打开一个 480x480 的 SDL2 窗口，显示：
- 炫彩渐变背景（鸿蒙深色风格）
- 动态 LOGO 动画
- 实时网络速度（模拟数据）
- 隧道数量统计
- 运行时间显示

## 架构说明

```
internal/display/
├── display.go          # 统一 Display 接口
├── display_sdl.go      # SDL2 实现（开发环境，build tag: preview）
├── display_fb.go       # Framebuffer 实现（生产环境，build tag: !preview）
├── graphics.go         # 图形绘制库（矩形、圆形、文本、渐变）
├── manager.go          # 显示管理器
└── status_page.go      # 实时状态页
```

### Build Tags

- `preview`: 开发环境（SDL2）
- 无 tag: 生产环境（Framebuffer）

## 已实现功能

### 核心模块
- [x] Display 接口抽象
- [x] SDL2 驱动（开发环境）
- [x] Framebuffer 驱动（生产环境）
- [x] 图形绘制库
  - [x] 矩形绘制
  - [x] 圆角矩形
  - [x] 圆形绘制
  - [x] 渐变绘制
  - [x] 文本渲染（8x8 位图字体）

### UI 功能
- [x] 实时状态页
- [x] 渐变背景（鸿蒙风格）
- [x] LOGO 动画
- [x] 数据卡片展示
- [x] 模拟数据更新

## 待实现功能

### UI 组件
- [ ] 按钮组件
- [ ] 列表组件
- [ ] 输入框组件
- [ ] 虚拟键盘

### 页面系统
- [ ] 设备设置页
- [ ] WiFi 列表页
- [ ] WiFi 连接页
- [ ] 隧道列表页
- [ ] 隧道编辑页

### 交互功能
- [ ] 触摸事件处理（SDL 鼠标 / GT911 evdev）
- [ ] 页面切换
- [ ] 30秒自动返回实时状态页

### 业务集成
- [ ] FRP 客户端数据绑定
- [ ] 网络管理器集成
- [ ] 配置管理集成

## 构建命令

```bash
# 查看帮助
make -f Makefile.display help

# 构建开发预览版本
make -f Makefile.display build-preview

# 运行预览
make -f Makefile.display run-preview

# 构建生产版本
make -f Makefile.display build-prod

# 交叉编译到 Linux ARM64
make -f Makefile.display build-linux-arm64

# 清理
make -f Makefile.display clean
```

## 技术细节

### 双缓冲渲染

使用 `image.RGBA` 作为离屏缓冲区，渲染完成后一次性刷新到显示设备，避免闪烁。

### 颜色方案（鸿蒙风格）

- 背景渐变: `#1A1A2E` → `#16213E` → `#0F3460`
- 主色调: `#667EEA` (蓝紫色)
- 强调色: `#00D4FF` (青色), `#2ED573` (绿色), `#FFC312` (黄色)

### 性能

- 目标帧率: 30 FPS
- 缓冲区大小: 480×480×4 = 921,600 字节 (~900KB)
- 数据更新频率: 2 秒

## 生产环境部署

### 硬件要求

- Luckfox Pico Ultra RV1106
- 480×480 RGB666 ISP 触摸屏
- GT911 电容触摸屏

### 部署步骤

1. 交叉编译到 ARM64:
   ```bash
   make -f Makefile.display build-linux-arm64
   ```

2. 传输到设备:
   ```bash
   scp bin/nwct-client-linux-arm64 root@device:/usr/local/bin/nwct-client
   ```

3. 运行（需要 root 权限访问 `/dev/fb0`）:
   ```bash
   sudo /usr/local/bin/nwct-client
   ```

## 开发说明

### 添加新页面

1. 在 `internal/display/` 创建页面文件（如 `settings_page.go`）
2. 实现 `Render()` 方法
3. 在 `Manager` 中添加页面切换逻辑

### 添加新 UI 组件

1. 在 `internal/display/ui/` 创建组件文件
2. 实现 `Component` 接口
3. 在页面中使用组件

## 截图

> TODO: 添加 SDL2 预览窗口截图

## 许可证

与主项目保持一致

