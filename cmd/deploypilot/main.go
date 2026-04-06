package main

import (
	"fmt"
	"os"
)

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("deploypilot version %s\n", version)
		return
	}
	fmt.Printf("deploypilot %s\n", version)
}
