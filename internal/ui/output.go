// Package ui provides styled terminal output for bragctl.
package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(16)
)

// Success prints a green checkmark + message to stdout.
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(successStyle.Render("✓ " + msg))
}

// Error prints a red message to stderr.
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, errorStyle.Render(msg))
}

// Info prints a blue message to stdout.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(infoStyle.Render(msg))
}

// Dim prints a gray/muted message to stdout.
func Dim(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(dimStyle.Render(msg))
}

// Bold prints a bold message to stdout.
func Bold(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(boldStyle.Render(msg))
}

// KeyValue prints a label-value pair with aligned formatting.
func KeyValue(label, value string) {
	fmt.Println(labelStyle.Render(label) + value)
}

// IsTerminal reports whether stdin is connected to a terminal.
func IsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
