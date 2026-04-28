package version

// Version is the current DeployPilot version.
// It is overridden at build time via -ldflags:
//   -X github.com/Yogdunana/deploypilot/internal/version.Version=x.y.z
var Version = "1.1.0"

// GitCommit is the git commit hash, set at build time.
var GitCommit = "unknown"

// BuildTime is the build timestamp, set at build time.
var BuildTime = "unknown"

// Info returns the complete version information.
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_time": BuildTime,
	}
}
