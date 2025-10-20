package version

var (
	// Should be overridden during the final release build with ldflags
	// to contain the actual version number
	BuildVersion = "devel"
)