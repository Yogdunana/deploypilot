package service

import "testing"

// TestParsePsAuxLine_Valid checks that a well-formed `ps aux` row is parsed
// into the expected ProcessInfo fields, including the KB→MB conversion
// performed for memory and virtual memory.
func TestParsePsAuxLine_Valid(t *testing.T) {
	// USER PID %CPU %MEM VSZ RSS STAT START TIME COMMAND
	line := "root      1234  2.5  1.0 1024000 524288 S   10:30   0:05 /usr/bin/process --flag"

	info := parsePsAuxLine(line)
	if info == nil {
		t.Fatalf("expected non-nil ProcessInfo for valid line: %q", line)
	}

	if info.PID != 1234 {
		t.Errorf("PID = %d, want 1234", info.PID)
	}
	if info.User != "root" {
		t.Errorf("User = %q, want %q", info.User, "root")
	}
	if info.CPU != 2.5 {
		t.Errorf("CPU = %v, want 2.5", info.CPU)
	}
	if info.State != "S" {
		t.Errorf("State = %q, want %q", info.State, "S")
	}
	// VSZ=1024000 KB → 1000 MB; RSS=524288 KB → 512 MB.
	if info.VirtualMem != 1000 {
		t.Errorf("VirtualMem = %v, want 1000 (KB→MB conversion)", info.VirtualMem)
	}
	if info.Memory != 512 {
		t.Errorf("Memory = %v, want 512 (KB→MB conversion)", info.Memory)
	}
	if info.Command != "/usr/bin/process --flag" {
		t.Errorf("Command = %q, want %q", info.Command, "/usr/bin/process --flag")
	}
}

// TestParsePsAuxLine_Malformed ensures garbage input returns nil rather than
// partially populated structs.
func TestParsePsAuxLine_Malformed(t *testing.T) {
	cases := []string{
		"",
		"not a process line",
		"only one field",
		"root 1234 2.5", // too few columns
	}
	for _, line := range cases {
		if info := parsePsAuxLine(line); info != nil {
			t.Errorf("expected nil for malformed line %q, got %+v", line, info)
		}
	}
}

// TestParsePsAux_AggregatesStats verifies that a multi-line `ps aux` blob
// produces both the expected process list and the right aggregate statistics,
// including state-bucket classification (Running / Sleeping / Zombie / Stopped).
func TestParsePsAux_AggregatesStats(t *testing.T) {
	// Mixed states: 1×R, 2×S, 1×Z, 1×T
	input := `root         1  0.0  0.1   1000   1024 S   Jan01   0:00 /sbin/init
www-data  100  0.5  1.0  100000  1024 R   10:30   0:01 nginx: worker
mysql     200  1.5  2.0  200000  2048 S   10:30   0:02 mysqld
zombie    300  0.0  0.0       0      0 Z   10:30   0:00 [defunct]
traced    400  0.0  0.0    2048    512 T   10:30   0:00 strace me
`

	processes, stats := parsePsAux(input)

	if len(processes) != 5 {
		t.Fatalf("expected 5 processes, got %d", len(processes))
	}
	if stats.Total != 5 {
		t.Errorf("stats.Total = %d, want 5", stats.Total)
	}
	if stats.Running != 1 {
		t.Errorf("stats.Running = %d, want 1 (state R)", stats.Running)
	}
	if stats.Sleeping != 2 {
		t.Errorf("stats.Sleeping = %d, want 2 (state S)", stats.Sleeping)
	}
	if stats.Zombie != 1 {
		t.Errorf("stats.Zombie = %d, want 1 (state Z)", stats.Zombie)
	}
	// Both D and T are bucketed into Stopped.
	if stats.Stopped != 1 {
		t.Errorf("stats.Stopped = %d, want 1 (state T)", stats.Stopped)
	}

	// Total memory: 1024KB + 1024KB + 2048KB + 0KB + 512KB = 4608KB = 4.5MB.
	wantMem := 4.5
	if stats.TotalMem != wantMem {
		t.Errorf("stats.TotalMem = %v, want %v", stats.TotalMem, wantMem)
	}

	// Total CPU: 0.0 + 0.5 + 1.5 + 0.0 + 0.0 = 2.0
	if stats.TotalCPU != 2.0 {
		t.Errorf("stats.TotalCPU = %v, want 2.0", stats.TotalCPU)
	}
}

// TestParsePsAux_EmptyAndBlankLines ensures empty input and input with only
// blank lines yield no processes and zeroed stats (no nil-deref on slice ops).
func TestParsePsAux_EmptyAndBlankLines(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"whitespace":  "   \n\n   \n",
		"newlines":    "\n\n\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			processes, stats := parsePsAux(input)
			if len(processes) != 0 {
				t.Errorf("expected 0 processes, got %d", len(processes))
			}
			if stats.Total != 0 || stats.Running != 0 || stats.Sleeping != 0 ||
				stats.Zombie != 0 || stats.Stopped != 0 {
				t.Errorf("expected all stats to be zero, got %+v", stats)
			}
		})
	}
}
