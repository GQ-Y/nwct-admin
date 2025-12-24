# FRP 客户端二进制文件

此目录用于存放嵌入的 frpc 二进制文件。

## 文件命名规则

二进制文件应按以下规则命名：
- Linux amd64: `frpc_linux_amd64`
- Linux arm64: `frpc_linux_arm64`
- Linux arm: `frpc_linux_arm`
- Darwin amd64: `frpc_darwin_amd64`
- Darwin arm64: `frpc_darwin_arm64`
- Windows amd64: `frpc_windows_amd64.exe`
- Windows arm64: `frpc_windows_arm64.exe`

## 获取二进制文件

从 FRP 官方 GitHub releases 下载对应平台的 frpc 二进制：
https://github.com/fatedier/frp/releases

下载后重命名并放置到此目录。

## 构建说明

在构建 Go 程序时，这些二进制文件会被 `go:embed` 嵌入到最终的可执行文件中。
程序启动时会自动解压到临时目录使用。

