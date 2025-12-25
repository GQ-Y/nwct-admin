//go:build linux && amd64

package frp

import "embed"

//go:embed assets/frpc_linux_amd64 assets/README.md
var embeddedAssets embed.FS

