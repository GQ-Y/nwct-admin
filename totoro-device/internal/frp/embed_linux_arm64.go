//go:build linux && arm64

package frp

import "embed"

//go:embed assets/frpc_linux_arm64 assets/README.md
var embeddedAssets embed.FS

