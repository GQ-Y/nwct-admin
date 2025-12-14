package version

// 通过 -ldflags 注入，未注入时为 dev
var (
	Version   = "dev"
	Commit    = ""
	BuildTime = ""
)


