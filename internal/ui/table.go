package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	borderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	runningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	stoppedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	defaultMarker = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("*")
)

// SiteRow holds display data for a single site in the list table.
type SiteRow struct {
	Name      string
	Engine    string
	AI        string
	Status    string
	Port      string
	IsDefault bool
}

// ContextRow holds display data for a context file.
type ContextRow struct {
	Name    string
	Enabled bool
}

// PrintContextTable renders a styled table of context files.
func PrintContextTable(rows []ContextRow) {
	data := make([][]string, len(rows))
	for i, r := range rows {
		status := stoppedStyle.Render("disabled")
		if r.Enabled {
			status = runningStyle.Render("enabled")
		}
		data[i] = []string{r.Name, status}
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Headers("Context", "Status").
		Rows(data...)

	fmt.Println(t)
}

// PrintKeyValueTable renders key-value pairs in a bordered box.
func PrintKeyValueTable(rows [][]string) {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(_, col int) lipgloss.Style {
			if col == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
			}
			return lipgloss.NewStyle()
		}).
		Rows(rows...)

	fmt.Println(t)
}

// PrintSiteTable renders a styled table of sites.
func PrintSiteTable(rows []SiteRow) {
	data := make([][]string, len(rows))
	for i, r := range rows {
		marker := " "
		if r.IsDefault {
			marker = defaultMarker
		}

		ai := r.AI
		if ai == "" {
			ai = "-"
		}

		status := r.Status
		if status == "running" {
			status = runningStyle.Render(status)
		} else {
			status = stoppedStyle.Render(status)
		}

		data[i] = []string{marker, r.Name, r.Engine, ai, status, r.Port}
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Headers("", "Site", "Engine", "AI", "Status", "Port").
		Rows(data...)

	fmt.Println(t)
}
