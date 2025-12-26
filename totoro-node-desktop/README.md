# totoro_node_desktop

A new Flutter project.

## Windows x64 构建（推荐）

macOS 无法直接交叉编译 Windows `.exe`。仓库已提供 GitHub Actions 工作流来构建 Windows x64 包：

- 工作流文件：`.github/workflows/totoro-node-desktop-windows-x64.yml`

使用方式：

1. 将改动 push 到你的仓库
2. 打开 GitHub → `Actions` → 选择 `totoro-node-desktop (windows-x64)` → `Run workflow`
3. 运行完成后在 `Artifacts` 下载 `totoro-node-desktop-windows-x64.zip`
4. 解压后运行 `totoro_node_desktop.exe`
