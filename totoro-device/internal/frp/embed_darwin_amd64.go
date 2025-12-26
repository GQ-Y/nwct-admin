//go:build darwin && amd64

package frp

import "embed"

//go:embed assets/frpc_darwin_amd64 assets/README.md
var embeddedAssets embed.FS

