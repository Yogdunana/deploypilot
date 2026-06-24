package mcp

import (
	"os"
	"testing"
)

// TestValidateVolumePath_AbsoluteAndUnderRoot verifies the happy path: an
// absolute path under one of the default allowed roots is accepted.
func TestValidateVolumePath_AbsoluteAndUnderRoot(t *testing.T) {
	// Ensure no custom root configuration affects this test.
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "")

	for _, p := range []string{"/app/data", "/opt/deploypilot", "/tmp/work", "/data/uploads"} {
		if err := validateVolumePath(p); err != nil {
			t.Errorf("validateVolumePath(%q) returned error: %v", p, err)
		}
	}
}

// TestValidateVolumePath_RejectsPathTraversal ensures ".." sequences are
// always rejected, which is critical to prevent a sandboxed volume mount
// from escaping to /etc, /root, or other sensitive host directories.
func TestValidateVolumePath_RejectsPathTraversal(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "")

	cases := []string{
		"/app/../etc/passwd",
		"/opt/../../etc/shadow",
		"/tmp/../root/.ssh/id_rsa",
		"/data/../etc/hosts",
		"/foo/../../../etc/passwd",
		"../etc/passwd", // even non-absolute paths with traversal are rejected
	}
	for _, p := range cases {
		if err := validateVolumePath(p); err == nil {
			t.Errorf("validateVolumePath(%q) should have been rejected for path traversal", p)
		}
	}
}

// TestValidateVolumePath_RejectsRelativePath ensures non-absolute paths are
// rejected even when no ".." is present.
func TestValidateVolumePath_RejectsRelativePath(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "")

	cases := []string{
		"app/data",
		"./data",
		"data/uploads",
		"etc/passwd",
	}
	for _, p := range cases {
		if err := validateVolumePath(p); err == nil {
			t.Errorf("validateVolumePath(%q) should have been rejected: relative path", p)
		}
	}
}

// TestValidateVolumePath_RejectsOutsideAllowedRoot ensures that even a clean
// absolute path outside the allowed root set is rejected.
func TestValidateVolumePath_RejectsOutsideAllowedRoot(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data")

	cases := []string{
		"/etc/passwd",
		"/root/.ssh/id_rsa",
		"/var/log/syslog",
		"/opt/private",
		"/tmp/secret",
	}
	for _, p := range cases {
		if err := validateVolumePath(p); err == nil {
			t.Errorf("validateVolumePath(%q) should be rejected (outside allowed roots)", p)
		}
	}
}

// TestValidateVolumePath_CustomRoots ensures custom roots are honored.
func TestValidateVolumePath_CustomRoots(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/srv/volumes:/home/user")

	if err := validateVolumePath("/srv/volumes/app1"); err != nil {
		t.Errorf("expected /srv/volumes/app1 to be allowed, got error: %v", err)
	}
	if err := validateVolumePath("/home/user/data"); err != nil {
		t.Errorf("expected /home/user/data to be allowed, got error: %v", err)
	}
	if err := validateVolumePath("/app/data"); err == nil {
		t.Errorf("expected /app/data to be rejected under custom roots")
	}
}

// TestExtractRegistry covers the registry extraction logic for various image
// name formats.
func TestExtractRegistry(t *testing.T) {
	cases := []struct {
		image string
		want  string
	}{
		{"nginx", "docker.io"},
		{"nginx:latest", "docker.io"},
		{"library/nginx", "docker.io"},
		{"myuser/myapp:v1.2", "docker.io"},
		{"ghcr.io/foo/bar:latest", "ghcr.io"},
		{"registry.gitlab.com/group/project", "registry.gitlab.com"},
		{"quay.io/prometheus/node-exporter", "quay.io"},
		{"localhost:5000/myapp", "localhost:5000"},
		{"127.0.0.1:5000/myapp", "127.0.0.1:5000"},
		{"gcr.io/google-project/image", "gcr.io"},
		{"private.registry.example.com:8443/team/svc", "private.registry.example.com:8443"},
	}
	for _, c := range cases {
		if got := extractRegistry(c.image); got != c.want {
			t.Errorf("extractRegistry(%q) = %q, want %q", c.image, got, c.want)
		}
	}
}

// TestValidateImageRegistry_NoWhitelist ensures that an empty whitelist
// allows any registry (so by default the platform is open). The
// function should return nil for any image when the whitelist is empty.
func TestValidateImageRegistry_NoWhitelist(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "")

	for _, image := range []string{
		"nginx:latest",
		"ghcr.io/owner/app:tag",
		"myregistry.example.com:5000/team/svc",
		"http://insecure.local/image",
	} {
		if err := validateImageRegistry(image); err != nil {
			t.Errorf("validateImageRegistry(%q) with no whitelist should allow image, got: %v", image, err)
		}
	}
}

// TestValidateImageRegistry_WhitelistEnforced ensures that a configured
// whitelist rejects images from registries that are not in the list.
func TestValidateImageRegistry_WhitelistEnforced(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "ghcr.io,docker.io,quay.io")

	allowed := []string{
		"nginx",                       // docker.io (default)
		"library/nginx:latest",        // docker.io
		"ghcr.io/owner/app:tag",       // direct match
		"quay.io/prometheus/alertmgr", // direct match
		// Subdomain match (ghcr.io is allowed, foo.ghcr.io should also pass)
		"sub.ghcr.io/foo/bar",
	}
	for _, image := range allowed {
		if err := validateImageRegistry(image); err != nil {
			t.Errorf("validateImageRegistry(%q) should be allowed, got: %v", image, err)
		}
	}

	disallowed := []string{
		"registry.gitlab.com/team/app",
		"myregistry.example.com/svc",
		"private.local/app",
		"evil.example.org/foo/bar",
	}
	for _, image := range disallowed {
		if err := validateImageRegistry(image); err == nil {
			t.Errorf("validateImageRegistry(%q) should be rejected, but got nil", image)
		}
	}
}

// TestValidateImageRegistry_WhitelistMatchSubdomain verifies the
// "subdomain-of-allowed" rule. An allowed registry `foo.com` should
// also accept `bar.foo.com`.
func TestValidateImageRegistry_WhitelistMatchSubdomain(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "foo.com")

	if err := validateImageRegistry("foo.com/team/app"); err != nil {
		t.Errorf("direct match should be allowed: %v", err)
	}
	if err := validateImageRegistry("region.foo.com/team/app"); err != nil {
		t.Errorf("subdomain of allowed registry should be allowed: %v", err)
	}
	if err := validateImageRegistry("notfoo.com/team/app"); err == nil {
		t.Errorf("look-alike domain should be rejected")
	}
}

// TestGetAllowedVolumeRoots_DefaultWhenUnset checks that the default roots
// are returned when the env var is missing/empty.
func TestGetAllowedVolumeRoots_DefaultWhenUnset(t *testing.T) {
	// Clear it explicitly; t.Setenv makes sure the value is restored after the test.
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "")
	roots := getAllowedVolumeRoots()
	if len(roots) == 0 {
		t.Fatal("expected default roots to be returned when env unset")
	}
	for _, r := range roots {
		if r == "" {
			t.Error("default roots must not contain empty strings")
		}
	}
}

// TestGetAllowedVolumeRoots_CustomSplit verifies that the colon-separated
// env var is split into individual roots.
func TestGetAllowedVolumeRoots_CustomSplit(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/srv/a:/srv/b:/srv/c")
	roots := getAllowedVolumeRoots()
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d (%v)", len(roots), roots)
	}
	want := []string{"/srv/a", "/srv/b", "/srv/c"}
	for i, w := range want {
		if roots[i] != w {
			t.Errorf("roots[%d] = %q, want %q", i, roots[i], w)
		}
	}
}

// TestGetAllowedRegistries_DefaultNilWhenUnset ensures the registry list is
// nil when no whitelist is configured.
func TestGetAllowedRegistries_DefaultNilWhenUnset(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "")
	got := getAllowedRegistries()
	if got != nil {
		t.Errorf("expected nil when env unset, got %v", got)
	}
}

// TestGetAllowedRegistries_CommaSplit ensures the comma-separated env var
// is split into individual registry entries (with no empty values).
func TestGetAllowedRegistries_CommaSplit(t *testing.T) {
	t.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "ghcr.io,docker.io,quay.io")
	got := getAllowedRegistries()
	if len(got) != 3 {
		t.Fatalf("expected 3 registries, got %d (%v)", len(got), got)
	}
	if got[0] != "ghcr.io" || got[1] != "docker.io" || got[2] != "quay.io" {
		t.Errorf("unexpected registry list: %v", got)
	}
}

// Sanity: ensure we don't accidentally introduce a dependency on host env.
var _ = os.Getenv
