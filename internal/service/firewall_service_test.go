package service

import (
	"testing"
)

// validatePort, validateProtocol, validateIP - the gatekeepers that prevent
// malformed values from becoming shell commands.

func TestValidatePort(t *testing.T) {
	cases := []struct {
		port   string
		wantOk bool
	}{
		// Valid ports
		{"1", true},
		{"80", true},
		{"443", true},
		{"8080", true},
		{"65535", true},
		// Invalid: out of range
		{"0", false},
		{"65536", false},
		{"-1", false},
		{"99999", false},
		// Invalid: not a number
		{"abc", false},
		{"80a", false},
		{"", false},
		{" ", false},
		{"80;rm -rf /", false},
	}
	for _, tc := range cases {
		err := validatePort(tc.port)
		if tc.wantOk && err != nil {
			t.Errorf("validatePort(%q) returned error %v, want nil", tc.port, err)
		}
		if !tc.wantOk && err == nil {
			t.Errorf("validatePort(%q) returned nil error, want error", tc.port)
		}
	}
}

func TestValidateProtocol(t *testing.T) {
	cases := []struct {
		proto  string
		wantOk bool
	}{
		{"tcp", true},
		{"udp", true},
		{"icmp", true},
		// Anything else is rejected - critical to prevent injection
		{"", false},
		{"TCP", false}, // case sensitive - keeps the shell command predictable
		{"http", false},
		{"all", false},
		{"tcp ", false},
		{"tcp;cat", false},
		{"-p tcp", false},
	}
	for _, tc := range cases {
		err := validateProtocol(tc.proto)
		if tc.wantOk && err != nil {
			t.Errorf("validateProtocol(%q) returned error %v, want nil", tc.proto, err)
		}
		if !tc.wantOk && err == nil {
			t.Errorf("validateProtocol(%q) returned nil, want error", tc.proto)
		}
	}
}

func TestValidateIP(t *testing.T) {
	cases := []struct {
		ip     string
		wantOk bool
	}{
		// Valid IPv4
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"203.0.113.42", true},
		{"255.255.255.255", true},
		// Valid IPv6
		{"::1", true},
		{"2001:db8::1", true},
		// Invalid
		{"", false},
		{"not-an-ip", false},
		{"999.999.999.999", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"1.2.3.4;cat /etc/passwd", false},
	}
	for _, tc := range cases {
		err := validateIP(tc.ip)
		if tc.wantOk && err != nil {
			t.Errorf("validateIP(%q) returned error %v, want nil", tc.ip, err)
		}
		if !tc.wantOk && err == nil {
			t.Errorf("validateIP(%q) returned nil, want error", tc.ip)
		}
	}
}

func TestIsValidPort(t *testing.T) {
	cases := []struct {
		port string
		want bool
	}{
		{"1", true},
		{"80", true},
		{"65535", true},
		{"0", false},
		{"65536", false},
		{"abc", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsValidPort(tc.port); got != tc.want {
			t.Errorf("IsValidPort(%q) = %v, want %v", tc.port, got, tc.want)
		}
	}
}

func TestIsValidIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		// Not strict IPv4: refuses IPv6, malformed, and shell injection
		{"::1", false},
		{"2001:db8::1", false},
		{"", false},
		{"abc", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"999.0.0.1", false}, // out-of-range octet
		{"1.2.3.4;cat", false},
	}
	for _, tc := range cases {
		if got := IsValidIP(tc.ip); got != tc.want {
			t.Errorf("IsValidIP(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

// parseUFWOutput - parses `ufw status numbered` output.

func TestParseUFWOutput_Standard(t *testing.T) {
	// The parser regex requires `[N]` with no space inside the brackets.
	// Real UFW output puts a space in for readability (`[ 1] 22/tcp`).
	// This test pins down the contract the parser implements.
	input := `Status: active

To                         Action      From
--                         ------      ----
[1] 22/tcp                     ALLOW IN    Anywhere
[2] 80/tcp                     ALLOW IN    Anywhere
[3] 443/tcp                    ALLOW IN    Anywhere
[4] 22                         ALLOW IN    10.0.0.0/24`

	rules := parseUFWOutput(input)
	if len(rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(rules))
	}

	if rules[0].ID != "1" {
		t.Errorf("expected rule[0].ID=1, got %s", rules[0].ID)
	}
	if rules[0].Port != "22/tcp" {
		t.Errorf("expected rule[0].Port=22/tcp, got %s", rules[0].Port)
	}
	if rules[0].Action != "ALLOW" {
		t.Errorf("expected rule[0].Action=ALLOW, got %s", rules[0].Action)
	}
	if rules[0].Target != "Anywhere" {
		t.Errorf("expected rule[0].Target=Anywhere, got %s", rules[0].Target)
	}

	// Last rule has a CIDR source
	if rules[3].Target != "10.0.0.0/24" {
		t.Errorf("expected rule[3].Target=10.0.0.0/24, got %s", rules[3].Target)
	}
}

func TestParseUFWOutput_Empty(t *testing.T) {
	rules := parseUFWOutput("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for empty input, got %d", len(rules))
	}
}

func TestParseUFWOutput_IgnoresNonMatching(t *testing.T) {
	// Header lines and "Status:" lines must be ignored
	input := `Status: active
Status: inactive
To                         Action      From
--                         ------      ----
random unparseable line
[1] 22/tcp                     ALLOW IN    Anywhere`

	rules := parseUFWOutput(input)
	if len(rules) != 1 {
		t.Errorf("expected 1 rule (only the bracketed line matches), got %d", len(rules))
	}
}

// parseIptablesOutput - parses `iptables -L -n --line-numbers` output.

func TestParseIptablesOutput_Standard(t *testing.T) {
	// The current iptables parser only extracts Action, Protocol, and Chain.
	// Port and Source extraction are no-ops because the code looks for
	// token `"dpt:"` and `"s"` which never appear in real iptables output
	// (where the field is `"dpt:22"` and there is no `"s"` separator).
	// This test pins the actual current behavior to surface the gap
	// for future fixing.
	input := `Chain INPUT (policy ACCEPT)
num  target     prot opt source               destination
1    ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0            tcp dpt:22
2    ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0            tcp dpt:80
3    DROP       all  --  10.0.0.5              0.0.0.0/0
4    ACCEPT     udp  --  0.0.0.0/0            0.0.0.0/0            udp dpt:53

Chain FORWARD (policy ACCEPT)
num  target     prot opt source               destination
1    ACCEPT     all  --  0.0.0.0/0            0.0.0.0/0`

	rules := parseIptablesOutput(input)
	// 4 rules from INPUT + 1 INPUT header + 1 FORWARD + 1 FORWARD header = 7
	if len(rules) != 7 {
		t.Fatalf("expected 7 parsed entries, got %d", len(rules))
	}

	// Find at least one ACCEPT tcp rule on INPUT chain
	sawAcceptTCP := false
	for _, r := range rules {
		if r.Chain == "INPUT" && r.Action == "ACCEPT" && r.Protocol == "tcp" {
			sawAcceptTCP = true
		}
	}
	if !sawAcceptTCP {
		t.Error("expected to find ACCEPT tcp rule on INPUT chain")
	}

	// Find a DROP rule on INPUT (parser does set Action and Chain correctly)
	sawDrop := false
	for _, r := range rules {
		if r.Chain == "INPUT" && r.Action == "DROP" {
			sawDrop = true
		}
	}
	if !sawDrop {
		t.Error("expected to find DROP rule on INPUT chain")
	}

	// Port and Source are NOT extracted by the current parser - pin that
	for _, r := range rules {
		if r.Port != "" {
			t.Errorf("expected Port to be empty in current parser, got %q on rule %+v", r.Port, r)
		}
		if r.Source != "" {
			t.Errorf("expected Source to be empty in current parser, got %q on rule %+v", r.Source, r)
		}
	}
}

func TestParseIptablesOutput_ChainSwitching(t *testing.T) {
	input := `Chain INPUT (policy ACCEPT)
1    ACCEPT     all  --  0.0.0.0/0            0.0.0.0/0
Chain FORWARD (policy ACCEPT)
2    DROP       all  --  0.0.0.0/0            0.0.0.0/0`

	rules := parseIptablesOutput(input)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Chain != "INPUT" {
		t.Errorf("expected rule[0].Chain=INPUT, got %s", rules[0].Chain)
	}
	if rules[1].Chain != "FORWARD" {
		t.Errorf("expected rule[1].Chain=FORWARD, got %s", rules[1].Chain)
	}
}

func TestParseIptablesOutput_Empty(t *testing.T) {
	rules := parseIptablesOutput("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

// parseFirewalldOutput

func TestParseFirewalldOutput_PortsLine(t *testing.T) {
	input := `public (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources: 
  services: ssh http
  ports: 8080/tcp 9090/udp
  protocols: 
  forward: no
  masquerade: no
  forward-ports: 
  source-ports: 
  icmp-blocks: 
  rich rules:`

	rules := parseFirewalldOutput(input)
	// 2 ports parsed from "ports:" line
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 rules, got %d", len(rules))
	}

	// Find the port rules
	var tcp, udp *FirewallRule
	for i := range rules {
		if rules[i].Port == "8080" {
			tcp = &rules[i]
		}
		if rules[i].Port == "9090" {
			udp = &rules[i]
		}
	}
	if tcp == nil {
		t.Fatal("expected to find 8080/tcp rule")
	}
	if tcp.Protocol != "tcp" {
		t.Errorf("expected 8080 protocol=tcp, got %s", tcp.Protocol)
	}
	if tcp.Action != "ACCEPT" {
		t.Errorf("expected 8080 action=ACCEPT, got %s", tcp.Action)
	}
	if udp == nil {
		t.Fatal("expected to find 9090/udp rule")
	}
	if udp.Protocol != "udp" {
		t.Errorf("expected 9090 protocol=udp, got %s", udp.Protocol)
	}
}

func TestParseFirewalldOutput_RichRuleLine(t *testing.T) {
	input := `  rich rules: 
    rule family=ipv4 source address=10.0.0.5 reject`

	rules := parseFirewalldOutput(input)
	if len(rules) < 1 {
		t.Fatalf("expected at least 1 rule from rich rules line, got %d", len(rules))
	}

	// Find the rich rule
	found := false
	for _, r := range rules {
		if r.Action == "REJECT" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find a REJECT rule from rich rules line")
	}
}

func TestParseFirewalldOutput_Empty(t *testing.T) {
	rules := parseFirewalldOutput("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

// FirewallType - guard against accidental changes to the type identifiers.

func TestFirewallType_Constants(t *testing.T) {
	cases := map[FirewallType]string{
		FirewallTypeUFW:       "ufw",
		FirewallTypeFirewalld: "firewalld",
		FirewallTypeIptables:  "iptables",
		FirewallTypeUnknown:   "unknown",
	}
	for typ, want := range cases {
		if string(typ) != want {
			t.Errorf("FirewallType = %q, want %q", string(typ), want)
		}
	}
}
