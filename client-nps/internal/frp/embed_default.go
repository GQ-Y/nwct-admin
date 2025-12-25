//go:build (!linux || (!arm64 && !arm && !amd64)) && (!darwin || (!arm64 && !amd64)) && (!windows || (!amd64 && !arm64))

package frp

import "embed"

//go:embed assets/README.md
var embeddedAssets embed.FS

