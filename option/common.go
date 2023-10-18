package option

import (
	"context"
	"errors"
	"os"

	"golang.org/x/exp/slog"
	"golang.org/x/term"

	"github.com/koolay/oras-sdk/display"
)

type contextKey int

// loggerKey is the associated key type for logger entry in context.
const loggerKey contextKey = iota

// Common option struct.
type Common struct {
	Debug   bool
	Verbose bool
	TTY     *os.File

	// [Preview] do not show progress output
	noTTY bool
}

// WithContext returns a new FieldLogger and an associated Context derived from ctx.
func (opts *Common) WithContext(ctx context.Context) (context.Context, *slog.Logger) {
	return display.NewLogger(ctx, true)
}

// Parse gets target options from user input.
func (opts *Common) Parse() error {
	// use STDERR as TTY output since STDOUT is reserved for pipeable output
	return opts.parseTTY(os.Stderr)
}

// parseTTY gets target options from user input.
func (opts *Common) parseTTY(f *os.File) error {
	if !opts.noTTY && term.IsTerminal(int(f.Fd())) {
		if opts.Debug {
			return errors.New("cannot use --debug, add --no-tty to suppress terminal output")
		}
		opts.TTY = f
	}
	return nil
}
