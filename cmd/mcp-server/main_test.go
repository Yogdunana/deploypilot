package main

import "testing"

func TestMCPVersionVariable(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}
}
