//go:build linux && arm && !arm64

package frp

import "embed"

//go:embed assets/frpc_linux_arm assets/README.md
var embeddedAssets embed.FS

