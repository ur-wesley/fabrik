// Package ui holds lipgloss styles and interactive confirms.
package ui

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Step renders "==> ..." section headers.
	Step = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	// OK renders success lines.
	OK = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	// Warn renders advisory lines.
	Warn = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	// Err renders failure lines.
	Err = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	// Dim renders hints.
	Dim = lipgloss.NewStyle().Faint(true)
)

// Confirm asks a yes/no question via huh. yes=true (from --yes/--non-interactive)
// skips the prompt. Non-TTY stdin defaults to yes so pipes never hang.
// Running under `go test` or CI automatically defaults to yes to avoid interactive hangs.
func Confirm(question string, yes bool) bool {
	if yes {
		return true
	}
	if flag.Lookup("test.v") != nil || os.Getenv("CI") != "" {
		return true
	}
	if f, err := os.Stdin.Stat(); err == nil && (f.Mode()&os.ModeCharDevice) == 0 {
		return true
	}
	confirm := true
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(question).Value(&confirm)))
	if err := form.Run(); err != nil {
		return false
	}
	return confirm
}

// SelectApps prompts the user to select which apps should be supported.
// If yes=true or non-interactive/CI/test/piped stdin, it returns defaults or all apps.
func SelectApps(defaults []string, yes bool) ([]string, error) {
	if yes || flag.Lookup("test.v") != nil || os.Getenv("CI") != "" {
		if len(defaults) > 0 {
			return defaults, nil
		}
		return []string{"cursor", "pi", "antigravity", "opencode"}, nil
	}
	if f, err := os.Stdin.Stat(); err == nil && (f.Mode()&os.ModeCharDevice) == 0 {
		if len(defaults) > 0 {
			return defaults, nil
		}
		return []string{"cursor", "pi", "antigravity", "opencode"}, nil
	}

	selected := defaults
	if len(selected) == 0 {
		selected = []string{"cursor", "pi", "antigravity", "opencode"}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which apps should Fabrik support?").
				Options(
					huh.NewOption("Cursor", "cursor"),
					huh.NewOption("Pi", "pi"),
					huh.NewOption("Antigravity", "antigravity"),
					huh.NewOption("OpenCode", "opencode"),
				).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("at least one app must be selected")
	}
	return selected, nil
}

// Header prints a blank line plus "==> text".
func Header(out *os.File, text string) {
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, Step.Render("==> "+text))
}

// Line prints a plain line.
func Line(out *os.File, text string) {
	fmt.Fprintln(out, text)
}
