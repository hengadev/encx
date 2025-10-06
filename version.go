package encx

import "fmt"

// Version of the encx library
const Version = "1.0.0"

// Build information (set by ldflags during build)
var (
	GitCommit string
	BuildDate string
	BuildUser string
)

// VersionInfo returns formatted version information
func VersionInfo() string {
	if GitCommit == "" {
		return fmt.Sprintf("encx v%s", Version)
	}
	return fmt.Sprintf("encx v%s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}

// FullVersionInfo returns complete version information including build user
func FullVersionInfo() VersionDetails {
	return VersionDetails{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
	}
}

// VersionDetails contains detailed version information
type VersionDetails struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	BuildUser string `json:"build_user,omitempty"`
}

// String returns a formatted version string
func (v VersionDetails) String() string {
	if v.GitCommit == "" {
		return fmt.Sprintf("v%s", v.Version)
	}
	return fmt.Sprintf("v%s-%s (%s)", v.Version, v.GitCommit[:7], v.BuildDate)
}
