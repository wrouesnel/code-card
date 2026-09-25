package code_card_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrouesnel/code-card/pkg/entrypoints/code_card"
	"github.com/wrouesnel/code-card/version"
	"github.com/wrouesnel/ctxstdio"
)

// runEntrypoint invokes the entrypoint as though from the command line and captures its output.
// Tests run in a temporary working directory.
func runEntrypoint(t *testing.T, stdIn io.Reader, args ...string) (int, string, string) {
	t.Helper()
	stdOut, stdErr := new(bytes.Buffer), new(bytes.Buffer)
	if stdIn == nil {
		stdIn = strings.NewReader("")
	}
	ctx := ctxstdio.Set(context.Background(), stdOut, stdErr, io.NopCloser(stdIn))
	exitCode := code_card.Entrypoint(ctx, args)
	return exitCode, stdOut.String(), stdErr.String()
}

// assertPDF checks that path contains a PDF.
func assertPDF(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Errorf("%s is not a PDF", path)
	}
}

func TestVersion(t *testing.T) {
	exitCode, stdOut, _ := runEntrypoint(t, nil, "--version")
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0", exitCode)
	}
	if !strings.Contains(stdOut, version.Version) {
		t.Errorf("stdout %q does not contain version %q", stdOut, version.Version)
	}
}

func TestRandomDefaultOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	exitCode, stdOut, stdErr := runEntrypoint(t, nil, "random")
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0: %s", exitCode, stdErr)
	}
	if stdOut != "" || stdErr != "" {
		t.Errorf("expected no output on success, got stdout %q, stderr %q", stdOut, stdErr)
	}
	assertPDF(t, "code-card.pdf")
}

func TestRandomWithTitle(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	exitCode, _, stdErr := runEntrypoint(t, nil, "--title", "Test", "--output-file", output, "random")
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0: %s", exitCode, stdErr)
	}
	assertPDF(t, output)
}

func TestRandomToStdout(t *testing.T) {
	t.Chdir(t.TempDir())
	exitCode, stdOut, _ := runEntrypoint(t, nil, "random", "--output-file", "-")
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0", exitCode)
	}
	if !strings.HasPrefix(stdOut, "%PDF-") {
		t.Errorf("stdout is not a PDF")
	}
}

func TestEncodedFromFlags(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	exitCode, _, stdErr := runEntrypoint(t, nil, "encoded", "--keyword", "word", "--code", "1234", "--output-file", output)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0: %s", exitCode, stdErr)
	}
	assertPDF(t, output)
}

func TestEncodedFromEnv(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	t.Setenv("CODE_CARD_OUTPUT_FILE", output)
	t.Setenv("CODE_CARD_ENCODED_SALT", "a-sufficiently-long-salt")
	t.Setenv("CODE_CARD_ENCODED_KEYWORD", "word")
	t.Setenv("CODE_CARD_ENCODED_CODE", "1234")
	exitCode, _, stdErr := runEntrypoint(t, nil, "encoded")
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0: %s", exitCode, stdErr)
	}
	assertPDF(t, output)
}

func TestEncodedFromStdin(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	exitCode, _, stdErr := runEntrypoint(t, strings.NewReader("word\n1234\n"), "encoded", "--output-file", output)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0: %s", exitCode, stdErr)
	}
	if !strings.Contains(stdErr, "Keyword: ") || !strings.Contains(stdErr, "Code: ") {
		t.Errorf("stderr %q does not contain prompts", stdErr)
	}
	assertPDF(t, output)
}

func TestEncodedInvalidFails(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	exitCode, _, stdErr := runEntrypoint(t, nil, "encoded", "--keyword", "word", "--code", "12345", "--output-file", output)
	if exitCode != 1 {
		t.Fatalf("exit code: got %d, want 1", exitCode)
	}
	if want := "code-card: error: the keyword must be at least as long as the code\n"; stdErr != want {
		t.Errorf("stderr: got %q, want %q", stdErr, want)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Errorf("output file should not be created on failure")
	}
}

func TestEncodedMissingInputFails(t *testing.T) {
	output := filepath.Join(t.TempDir(), "card.pdf")
	if exitCode, _, _ := runEntrypoint(t, nil, "encoded", "--output-file", output); exitCode != 1 {
		t.Fatalf("exit code: got %d, want 1", exitCode)
	}
}

func TestInvalidFlagFails(t *testing.T) {
	exitCode, _, stdErr := runEntrypoint(t, nil, "--no-such-flag")
	if exitCode == 0 {
		t.Fatal("exit code: got 0, want non-zero")
	}
	if !strings.Contains(stdErr, "no-such-flag") {
		t.Errorf("stderr %q does not mention the bad flag", stdErr)
	}
}
