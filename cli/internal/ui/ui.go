// Package ui holds lipgloss styles and interactive confirms.
package ui

import (
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
func Confirm(question string, yes bool) bool {
	if yes {
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

// Header prints a blank line plus "==> text".
func Header(out *os.File, text string) {
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, Step.Render("==> "+text))
}

// Line prints a plain line.
func Line(out *os.File, text string) {
	fmt.Fprintln(out, text)
}
