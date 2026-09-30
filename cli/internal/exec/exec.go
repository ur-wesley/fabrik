// Package exec abstracts process execution for testability.
package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner runs external commands.
type Runner interface {
	// LookPath reports whether name resolves on PATH.
	LookPath(name string) (string, error)
	// Run executes name with args in the current directory, returning stdout.
	Run(ctx context.Context, name string, args ...string) (string, error)
	// RunIn executes name with args in dir, returning stdout.
	RunIn(ctx context.Context, dir, name string, args ...string) (string, error)
}

// OSRunner is the production Runner.
type OSRunner struct{}

// LookPath delegates to exec.LookPath.
func (OSRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

// Run executes the command and returns stdout (error includes stderr tail).
// Stdin is /dev/null so a prompting child fails fast instead of hanging;
// callers that need interactivity must handle it themselves.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return OSRunner{}.RunIn(ctx, "", name, args...)
}

// RunIn executes the command in dir.
func (OSRunner) RunIn(ctx context.Context, dir, name string, args ...string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		return "", err
	}
	defer null.Close()
	cmd.Stdin = null
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(out.String()), &Error{Name: name, Msg: msg, Err: err}
	}
	return strings.TrimSpace(out.String()), nil
}

// Error is a command failure with stderr context.
type Error struct {
	Name string
	Msg  string
	Err  error
}

func (e *Error) Error() string { return e.Name + ": " + e.Msg }
func (e *Error) Unwrap() error { return e.Err }

// Fake is a scripted Runner for tests.
type Fake struct {
	// Path contains names considered present on PATH.
	Path map[string]bool
	// Outputs maps "name args" to stdout.
	Outputs map[string]string
	// Errors maps "name args" to failure.
	Errors map[string]error
	// Calls records invocations as "name args".
	Calls []string
	// Dirs records the working dir per call (parallel to Calls).
	Dirs []string
	// Passthrough, when true, falls back to the real OS for unscripted commands.
	Passthrough bool
}

func (f *Fake) key(name string, args []string) string {
	k := name
	for _, a := range args {
		k += " " + a
	}
	return k
}

// LookPath reports scripted PATH presence.
func (f *Fake) LookPath(name string) (string, error) {
	if f.Path[name] {
		return "/fake/bin/" + name, nil
	}
	if f.Passthrough {
		return OSRunner{}.LookPath(name)
	}
	return "", &Error{Name: name, Msg: "not on PATH"}
}

// Run records the call and returns the scripted result.
func (f *Fake) Run(_ context.Context, name string, args ...string) (string, error) {
	return f.RunIn(context.Background(), "", name, args...)
}

// RunIn records the call with dir prefix and returns the scripted result.
func (f *Fake) RunIn(_ context.Context, dir, name string, args ...string) (string, error) {
	k := f.key(name, args)
	f.Calls = append(f.Calls, k)
	f.Dirs = append(f.Dirs, dir)
	if err, ok := f.Errors[k]; ok {
		return f.Outputs[k], err
	}
	if out, ok := f.Outputs[k]; ok {
		return out, nil
	}
	if f.Passthrough {
		return OSRunner{}.Run(context.Background(), name, args...)
	}
	return "", &Error{Name: name, Msg: "unscripted call: " + k}
}

// Called reports whether a call prefix was invoked.
func (f *Fake) Called(prefix string) bool {
	for _, c := range f.Calls {
		if c == prefix || strings.HasPrefix(c, prefix+" ") || strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}
