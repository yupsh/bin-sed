package main

import (
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-sed"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `sed SCRIPT [FILE...]

Apply the s/pattern/replacement/[flags] SCRIPT to each FILE.
When no FILE is specified, read from standard input.`

// Error is the sentinel error type for the yup-sed wrapper. It lets every
// error path this package raises be matched with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingScript is returned when no SCRIPT operand is supplied.
const ErrMissingScript Error = "missing SCRIPT operand"

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags while still exposing
// the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the sed CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newCommand(version, stdin, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "sed: %v\n", err)
		return 1
	}
	return 0
}

func newCommand(version string, stdin io.Reader, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "sed",
		Version:         version,
		Usage:           "stream editor for filtering and transforming text",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Action:         action(stdin, stdout, fs),
	}
}

func action(stdin io.Reader, stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		if c.NArg() == 0 {
			return ErrMissingScript
		}
		script := c.Args().Get(0)
		_, err := gloo.Run(source(c, stdin, fs), gloo.ByteWriteTo(stdout), command.Sed(script))
		return err
	}
}

func source(c *cli.Command, stdin io.Reader, fs afero.Fs) any {
	if c.NArg() <= 1 {
		return gloo.ByteReaderSource([]io.Reader{stdin})
	}
	names := c.Args().Slice()[1:]
	files := make([]gloo.File, len(names))
	for i, name := range names {
		files[i] = gloo.File(name)
	}
	return gloo.ByteFileSource(fs, files)
}
