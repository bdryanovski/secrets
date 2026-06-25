package version

// These variables are set at build time via ldflags.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// String returns a formatted version string.
func String() string {
	return Version + " (commit: " + GitCommit + ", built: " + BuildDate + ")"
}
