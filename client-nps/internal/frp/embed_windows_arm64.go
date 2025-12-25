//go:build windows && arm64

package frp

import "embed"

//go:embed assets/frpc_windows_arm64.exe assets/README.md
var embeddedAssets embed.FS

