//go:build windows && amd64

package frp

import "embed"

//go:embed assets/frpc_windows_amd64.exe assets/README.md
var embeddedAssets embed.FS

