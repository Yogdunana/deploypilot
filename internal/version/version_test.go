package version

import "testing"

func TestVersionIsSet(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestGitCommitIsSet(t *testing.T) {
	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}

func TestBuildTimeIsSet(t *testing.T) {
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestInfoReturnsAllFields(t *testing.T) {
	info := Info()
	if info["version"] == "" {
		t.Error("Info() should return non-empty version")
	}
	if info["git_commit"] == "" {
		t.Error("Info() should return non-empty git_commit")
	}
	if info["build_time"] == "" {
		t.Error("Info() should return non-empty build_time")
	}
}
