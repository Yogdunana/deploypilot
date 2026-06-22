package service

import (
	"testing"
)

// validateScript - the safety net that stops an operator (or compromised
// session) from running rm -rf /, mkfs, fork bombs, etc. through the
// toolbox "run script" endpoint. A regression here would let a single
// HTTP request destroy a production host.

func TestValidateScript_SafeCommands(t *testing.T) {
	tb := &ToolboxService{}
	safe := []string{
		"ls -la",
		"cat /etc/hosts",
		"df -h",
		"ps aux | grep nginx",
		"systemctl status nginx",
		"tail -n 100 /var/log/nginx/access.log",
		"echo hello",
		"uname -a",
		"free -h",
		"ss -tlnp",
		"curl -s http://example.com",
		"systemctl is-active nginx",
	}
	for _, s := range safe {
		if err := tb.validateScript(s); err != nil {
			t.Errorf("validateScript(%q) = %v, want nil (safe command)", s, err)
		}
	}
}

// TestValidateScript_KnownOverblocks pins the over-broad matching of the
// dangerous-pattern regex. The current implementation does substring
// matching against patterns like "rm\\s+-rf\\s+/" so it incorrectly
// blocks "rm -rf /tmp/build" and "echo 'rm -rf /' > warning.txt".
// A future fix to anchor the patterns (or use a proper AST check) is
// expected to flip these from "blocked" to "allowed", at which point
// this test should be updated to assert the new contract.
func TestValidateScript_KnownOverblocks(t *testing.T) {
	tb := &ToolboxService{}
	cases := []string{
		"echo 'rm -rf /' > warning.txt",
		"rm -rf /tmp/build",
		"rm -rf /home/user/old",
		"chmod 777 /tmp/shared",
	}
	for _, s := range cases {
		if err := tb.validateScript(s); err == nil {
			t.Errorf("validateScript(%q) = nil, want error (over-blocker; flip expectation when fixed)", s)
		}
	}
}

func TestValidateScript_DangerousCommands(t *testing.T) {
	tb := &ToolboxService{}
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*", // wildcard on root
		"rm -rf / --no-preserve-root",
		"rm    -rf    /",      // extra whitespace
		"mkfs.ext4 /dev/sda1", // filesystem create
		"mkfs.xfs /dev/sdb",
		"dd if=/dev/zero of=/dev/sda", // raw disk write
		"dd if=/dev/urandom of=/dev/sdc bs=1M",
		"> /dev/sda",  // direct write to disk
		">  /dev/sdb", // extra whitespace
		"chmod 777 /", // world-writable root
		"shutdown -h now",
		"reboot",
		"init 0",
		":(){ :|:& };:",        // fork bomb
		"echo hello; rm -rf /", // chained rm
		"foo\nrm -rf /",        // newline injection
	}
	for _, s := range dangerous {
		if err := tb.validateScript(s); err == nil {
			t.Errorf("validateScript(%q) = nil, want error (dangerous command)", s)
		}
	}
}

func TestValidateScript_EmptyInput(t *testing.T) {
	tb := &ToolboxService{}
	if err := tb.validateScript(""); err != nil {
		t.Errorf("empty script should be allowed (validation is for content, not size), got: %v", err)
	}
}

func TestValidateScript_WhitespaceOnly(t *testing.T) {
	tb := &ToolboxService{}
	whitespace := []string{
		"   ",
		"\n\n",
		"\t",
		" \t\n ",
	}
	for _, s := range whitespace {
		if err := tb.validateScript(s); err != nil {
			t.Errorf("whitespace-only script (%q) should be allowed, got: %v", s, err)
		}
	}
}

// GetBuiltInScript - guards against accidental deletion of common
// operator tools. These entries are referenced by ID from the UI/API,
// so renaming or removing them is a contract break.

func TestGetBuiltInScript_NetworkCategory(t *testing.T) {
	tb := &ToolboxService{}
	entries := map[string]string{
		"network/ping":        "ping",
		"network/dns":         "nslookup",
		"network/ports":       "ss",
		"network/connections": "ss",
		"network/firewall":    "iptables",
		"network/trace":       "traceroute",
	}
	for key, mustContain := range entries {
		script, err := tb.GetBuiltInScript(splitCategory(key), splitName(key))
		if err != nil {
			t.Errorf("GetBuiltInScript(%q) = error %v, want success", key, err)
			continue
		}
		if !contains(script, mustContain) {
			t.Errorf("GetBuiltInScript(%q) = %q, want substring %q", key, script, mustContain)
		}
	}
}

func TestGetBuiltInScript_NotFound(t *testing.T) {
	tb := &ToolboxService{}
	cases := []struct {
		cat, name string
	}{
		{"network", "nonexistent"},
		{"unknown", "ping"},
		{"", ""},
		{"NETWORK/ping", "ping"}, // case-sensitive categories
	}
	for _, c := range cases {
		if _, err := tb.GetBuiltInScript(c.cat, c.name); err == nil {
			t.Errorf("GetBuiltInScript(%q, %q) = nil error, want error", c.cat, c.name)
		}
	}
}

// splitCategory and splitName are tiny helpers for test readability.

func splitCategory(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[:i]
		}
	}
	return key
}

func splitName(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			return key[i+1:]
		}
	}
	return ""
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 ||
		(len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
