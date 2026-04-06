package main

import "testing"

func TestVersionVariable(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}
}
