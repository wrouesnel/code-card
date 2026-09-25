package code_card

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/wrouesnel/ctxstdio"
	"golang.org/x/term"
)

var errEmptyValue = errors.New("no value entered")

// prompter reads secret values from the context's stdin. Prompts are written to stderr so stdout
// remains usable for card output.
type prompter struct {
	out      io.Writer
	terminal *os.File
	lines    *bufio.Reader
}

func newPrompter(ctx context.Context) *prompter {
	stdIn := ctxstdio.Stdin(ctx)
	p := &prompter{out: ctxstdio.StdErr(ctx)}
	if f, ok := stdIn.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec // fd fits in int
		p.terminal = f
	} else {
		p.lines = bufio.NewReader(stdIn)
	}
	return p
}

// Secret prompts for a value. On a terminal input is hidden and must be entered twice to
// confirm it. Otherwise a single line is read from stdin.
func (p *prompter) Secret(name string) (string, error) {
	if p.terminal == nil {
		_, _ = fmt.Fprintf(p.out, "%s: ", name)
		line, err := p.lines.ReadString('\n')
		_, _ = fmt.Fprintln(p.out)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if err != nil {
				return "", errors.Wrap(err, "reading input")
			}
			return "", errEmptyValue
		}
		return line, nil
	}

	for {
		value, err := p.readHidden(name + ": ")
		if err != nil {
			return "", err
		}
		if value == "" {
			continue
		}
		confirm, err := p.readHidden("Repeat for confirmation: ")
		if err != nil {
			return "", err
		}
		if value == confirm {
			return value, nil
		}
		_, _ = fmt.Fprintln(p.out, "Error: The two entered values do not match.")
	}
}

func (p *prompter) readHidden(prompt string) (string, error) {
	_, _ = io.WriteString(p.out, prompt)
	value, err := term.ReadPassword(int(p.terminal.Fd())) //nolint:gosec // fd fits in int
	_, _ = fmt.Fprintln(p.out)
	if err != nil {
		return "", errors.Wrap(err, "reading input")
	}
	return string(value), nil
}
