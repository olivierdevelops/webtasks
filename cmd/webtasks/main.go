// webtasks is a long-running browser-automation server. The binary itself
// ships no configs — every deployment provides a "bundle" (directory or zip)
// of YAML/JS files at startup via WEBTASKS_BUNDLE.
package main

import (
	"fmt"
	"os"

	"webtasks/internal/orchestrator"
)

func main() {
	if err := orchestrator.Run(orchestrator.FromEnv(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "[webtasks] fatal:", err)
		os.Exit(1)
	}
}
