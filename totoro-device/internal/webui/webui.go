package webui

import (
	"embed"
	"io/fs"
)

// 注意：go:embed 必须在编译时匹配到至少一个文件。
// 我们在 dist/ 下提供一个最小 index.html 占位文件，保证未先 build 前端时也能 go build。

//go:embed dist/* dist/assets/*
var embedded embed.FS

// DistFS 返回以 dist 为根的只读文件系统（用于 http.FileServer/http.FS）。
func DistFS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		// 理论上不应发生：dist 目录应始终存在（至少包含占位 index.html）
		return embedded
	}
	return sub
}
