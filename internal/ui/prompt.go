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
