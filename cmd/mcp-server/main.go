package main

import (
	"fmt"
	"os"
)

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	fmt.Printf("mcp-server %s (placeholder)\n", version)
	os.Exit(0)
}
