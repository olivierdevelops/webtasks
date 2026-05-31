// Package console reads runtime values from the controlling terminal. Used by
// the secrets loader to ask for required secrets that weren't supplied via
// env or args. Writes prompts to stderr so they don't pollute stdout JSON.
package console

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type Prompter struct{}

// HasTTY reports whether stdin is attached to a terminal.
func (Prompter) HasTTY() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// ReadLine prompts on stderr and reads a non-secret line from stdin.
func (Prompter) ReadLine(msg string) (string, error) {
	fmt.Fprint(os.Stderr, msg)
	r := bufio.NewReader(os.Stdin)
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

// ReadPassword reads a line without echoing it. Falls back to echoed input
// when stdin isn't a TTY, with a stderr warning so the operator notices.
func (p Prompter) ReadPassword(msg string) (string, error) {
	if !p.HasTTY() {
		fmt.Fprintln(os.Stderr, "[webtasks] WARN: no TTY for silent input; secret may be visible.")
		return p.ReadLine(msg)
	}
	fmt.Fprint(os.Stderr, msg)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
