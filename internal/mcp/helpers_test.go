package mcp

import "testing"

func TestValidateVolumePath(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// happy paths (default allowed roots: /app /data /opt /tmp)
		{name: "root itself", path: "/app", wantErr: false},
		{name: "under root", path: "/app/data", wantErr: false},
		{name: "deep under root", path: "/opt/deploypilot/logs", wantErr: false},
		{name: "redundant separators are cleaned", path: "/data//nested/./x", wantErr: false},

		// rejections
		{name: "empty", path: "", wantErr: true},
		{name: "relative", path: "app/data", wantErr: true},
		{name: "traversal", path: "/app/../../etc", wantErr: true},
		{name: "outside allowed roots", path: "/etc/passwd", wantErr: true},
		{name: "sibling that shares a prefix", path: "/approot", wantErr: true},
		{name: "sibling with dash", path: "/tmp-evil", wantErr: true},
		{name: "sibling under home", path: "/home/user", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVolumePath(tc.path)
			if tc.wantErr && err == nil {
				t.Errorf("validateVolumePath(%q) = nil, want error", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateVolumePath(%q) = %v, want nil", tc.path, err)
			}
		})
	}
}

// The host path is a POSIX path even when DeployPilot runs on Windows, so the
// checks must not depend on the host OS. This is what broke the deploy handler
// on Windows: filepath.IsAbs("/app") is false there.
func TestValidateVolumePath_IsOSIndependent(t *testing.T) {
	if err := validateVolumePath("/app"); err != nil {
		t.Errorf(`validateVolumePath("/app") = %v, want nil regardless of host OS`, err)
	}
}
