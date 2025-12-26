//go:build darwin && arm64

package frp

import "embed"

//go:embed assets/frpc_darwin_arm64 assets/README.md
var embeddedAssets embed.FS

