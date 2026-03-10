package ui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
)

// PromptInput asks the user for text input with a default value pre-filled.
func PromptInput(title, defaultVal string) string {
	value := defaultVal
	err := huh.NewInput().
		Title(title).
		Value(&value).
		Run()
	if err != nil {
		return defaultVal
	}
	return value
}

// PromptSelect asks the user to choose from a list of options.
// Returns the selected value, or defaultVal on error/cancel.
func PromptSelect(title string, options []string, defaultVal string) string {
	value := defaultVal
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&value).
		Run()
	if err != nil {
		return defaultVal
	}
	return value
}

// PromptConfirm displays a message and waits for the user to press Enter.
// Returns an error if the user cancels (e.g. Ctrl-C).
func PromptConfirm(msg string) error {
	var confirmed bool
	return huh.NewConfirm().
		Title(msg).
		Affirmative("Continue").
		Negative("").
		Value(&confirmed).
		Run()
}

// PromptMultiSelect asks the user to select multiple items from a list.
// All items are selected by default. Returns the selected values.
// Returns an error if the user cancels (e.g. Ctrl-C).
func PromptMultiSelect(title string, options []string) ([]string, error) {
	selected := make([]string, len(options))
	copy(selected, options)

	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	err := huh.NewMultiSelect[string]().
		Title(title).
		Options(opts...).
		Value(&selected).
		Run()
	return selected, err
}

// PromptInputE is like PromptInput but returns an error on cancel.
func PromptInputE(title, defaultVal string) (string, error) {
	value := defaultVal
	err := huh.NewInput().
		Title(title).
		Value(&value).
		Run()
	return value, err
}

// Spin runs fn while showing a spinner with the given title.
func Spin(title string, fn func() error) error {
	var fnErr error
	if err := spinner.New().
		Title(title).
		Action(func() { fnErr = fn() }).
		Run(); err != nil {
		return err
	}
	return fnErr
}
