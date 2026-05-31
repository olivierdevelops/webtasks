// webtasks is a browser-automation engine. It can run as a long-lived HTTP
// server (the default) or operate on a single recipe from the command line.
//
//	webtasks                      start the server (bundle from WEBTASKS_BUNDLE)
//	webtasks serve                same as above
//	webtasks init [dir]           scaffold a starter bundle to build from
//	webtasks run <file>           run one .webtask/.yaml file once, print JSON
//	webtasks bundle <dir> [out]   package a directory into a runnable bundle zip
//
// Every deployment provides a "bundle" (directory or zip) of recipes/JS/config.
package main

import (
	"fmt"
	"os"
	"strings"

	"webtasks/internal/orchestrator"
)

func main() {
	args := os.Args[1:]

	// Default (no subcommand, or leading flags like `--SECRET=value`) starts
	// the server and forwards args to the secrets loader.
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			usage()
			return
		}
		fail(orchestrator.Run(orchestrator.FromEnv(), args))
		return
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "serve":
		fail(orchestrator.Run(orchestrator.FromEnv(), rest))
	case "init":
		fail(orchestrator.InitProject(rest))
	case "run":
		fail(orchestrator.RunFile(rest))
	case "bundle":
		fail(orchestrator.BundleDir(rest))
	case "version", "--version":
		fmt.Println("webtasks dev")
	case "help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		usage()
		os.Exit(2)
	}
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "[webtasks] fatal:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`webtasks — browser automation behind one API

Usage:
  webtasks                          start the HTTP server (bundle from WEBTASKS_BUNDLE)
  webtasks serve                    start the HTTP server
  webtasks init [dir]               scaffold a starter bundle to build from
  webtasks run <file> [opts]        run one recipe once and print JSON output
  webtasks bundle <dir> [out.zip]   package a directory into a runnable bundle
  webtasks version                  print version

run options:
  --input k=v        set one input (repeatable)
  --json '{...}'     set inputs from a JSON object

Files:
  .webtask           recipe source (transpiled via the bundled grammar; needs the capy CLI)
  .yaml              task definition (runs with no extra tools)

Environment (server + run):
  WEBTASKS_BUNDLE        bundle directory or .zip (server)
  WEBTASKS_HOST/PORT     server bind address (default 127.0.0.1:8765)
  WEBTASKS_HEADLESS      "true"/"false" (run defaults to headless)
  WEBTASKS_DOWNLOADS_DIR where downloaded files land
`)
}
