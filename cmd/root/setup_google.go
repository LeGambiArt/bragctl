package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/bragctl/bragctl/internal/config"
	"gitlab.cee.redhat.com/bragctl/bragctl/internal/ui"
	"gitlab.cee.redhat.com/bragctl/oauth2flow"
)

// googleService describes a Google API service for the setup wizard.
type googleService struct {
	Name      string
	TokenFile string
	Scopes    []string
}

var googleServices = []googleService{
	{
		Name:      "Calendar",
		TokenFile: "token-calendar.json",
		Scopes:    []string{"https://www.googleapis.com/auth/calendar"},
	},
	{
		Name:      "Gmail",
		TokenFile: "token-gmail.json",
		Scopes:    []string{"https://www.googleapis.com/auth/gmail.modify"},
	},
	{
		Name:      "Drive",
		TokenFile: "token-drive.json",
		Scopes:    []string{"https://www.googleapis.com/auth/drive.readonly"},
	},
}

func setupGoogleCmd() *cobra.Command {
	var credentialsFile string

	cmd := &cobra.Command{
		Use:   "google",
		Short: "Set up Google Calendar, Gmail, and Drive access",
		Long: `Interactive wizard to configure Google API credentials.

Guides you through creating a Google Cloud project, enabling APIs,
configuring OAuth consent, and authorizing access to Calendar, Gmail,
and Drive.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !ui.IsTerminal() {
				return fmt.Errorf("setup requires an interactive terminal")
			}
			return runGoogleSetup(cmd.Context(), credentialsFile)
		},
	}

	cmd.Flags().StringVar(&credentialsFile, "credentials", "",
		"Path to client credentials JSON (skip interactive steps 1-4)")

	return cmd
}

func runGoogleSetup(ctx context.Context, credentialsFile string) error {
	fmt.Println()
	ui.Bold("Google API Setup")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println()
	ui.Info("This sets up Google Calendar, Gmail, and Drive access.")
	ui.Dim("You'll need a Google Cloud project with OAuth credentials.")
	fmt.Println()

	credDir := googleCredentialsDir()
	clientCreds := filepath.Join(credDir, "client-credentials.json")

	if credentialsFile != "" {
		// Skip interactive steps, just copy the credentials file
		if err := copyCredentialsFile(credentialsFile, clientCreds); err != nil {
			return err
		}
	} else if _, err := os.Stat(clientCreds); err == nil {
		ui.Success("Client credentials already exist at %s", clientCreds)
		fmt.Println()
	} else {
		if err := runGoogleConsoleSteps(clientCreds); err != nil {
			return err
		}
	}

	// Step 5: Authorize services
	fmt.Println()
	ui.Bold("Authorize services")
	fmt.Println()

	serviceNames := make([]string, len(googleServices))
	for i, s := range googleServices {
		serviceNames[i] = s.Name
	}
	selected, err := ui.PromptMultiSelect("Which services to authorize?", serviceNames)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		ui.Dim("No services selected, skipping authorization.")
		return nil
	}

	fmt.Println()

	for _, svc := range googleServices {
		if !contains(selected, svc.Name) {
			continue
		}

		tokenPath := filepath.Join(credDir, svc.TokenFile)

		// Skip if token already exists
		if _, err := os.Stat(tokenPath); err == nil {
			ui.Success("%s already authorized", svc.Name)
			continue
		}

		ui.Info("Opening browser for %s consent...", svc.Name)

		flowCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		_, err := oauth2flow.Run(flowCtx, oauth2flow.Config{
			ClientCredentialsFile: clientCreds,
			TokenFile:             tokenPath,
			Scopes:                svc.Scopes,
		})
		cancel()

		if err != nil {
			return fmt.Errorf("authorize %s: %w", svc.Name, err)
		}
		ui.Success("%s authorized", svc.Name)
	}

	fmt.Println()
	ui.Success("Google setup complete")
	ui.KeyValue("Tokens:", credDir)
	fmt.Println()
	ui.Dim("To start using: bragctl ai")

	return nil
}

func runGoogleConsoleSteps(clientCreds string) error {
	// Step 1: Create project
	ui.Bold("Step 1: Create or select a Google Cloud project")
	openBrowserStep("https://console.cloud.google.com/projectcreate")
	if err := ui.PromptConfirm("Press Enter when your project is ready"); err != nil {
		return err
	}

	// Step 2: Enable APIs
	fmt.Println()
	ui.Bold("Step 2: Enable required APIs")
	openBrowserStep("https://console.cloud.google.com/apis/enableflow?apiid=calendar-json.googleapis.com,gmail.googleapis.com,drive.googleapis.com")
	if err := ui.PromptConfirm("Press Enter when APIs are enabled"); err != nil {
		return err
	}

	// Step 3: OAuth consent screen
	fmt.Println()
	ui.Bold("Step 3: Configure OAuth consent screen")
	ui.Dim("  In the Auth section (left panel):")
	ui.Dim("  - Branding: set app name and support email")
	ui.Dim("  - Audience: choose \"External\"")
	ui.Dim("    (if app is in Testing mode, add your email as test user)")
	ui.Dim("  If already configured, you can skip this step.")
	openBrowserStep("https://console.cloud.google.com/auth/overview")
	if err := ui.PromptConfirm("Press Enter when consent screen is configured"); err != nil {
		return err
	}

	// Step 4: Create credentials
	fmt.Println()
	ui.Bold("Step 4: Create OAuth client")
	ui.Dim("  In the Auth section (left panel):")
	ui.Dim("  - Clients > Create client")
	ui.Dim("  - Choose \"Desktop application\"")
	ui.Dim("  - Download the JSON file")
	openBrowserStep("https://console.cloud.google.com/auth/clients")

	path, err := ui.PromptInputE("Path to downloaded credentials JSON", "")
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("credentials file path is required")
	}

	return copyCredentialsFile(path, clientCreds)
}

func openBrowserStep(url string) {
	if err := oauth2flow.OpenBrowser(url); err != nil {
		// Browser failed to open — show the URL so the user can copy it
		ui.Dim("  Open: %s", url)
	} else {
		ui.Info("  Opening %s", url)
	}
	fmt.Println()
}

func copyCredentialsFile(src, dst string) error {
	// Expand ~ if present
	if strings.HasPrefix(src, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("expand home dir: %w", err)
		}
		src = filepath.Join(home, src[2:])
	}

	data, err := os.ReadFile(src) //nolint:gosec // user-provided credential file path
	if err != nil {
		return fmt.Errorf("read credentials file: %w", err)
	}

	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}

	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	ui.Success("Saved to %s", dst)
	return nil
}

func googleCredentialsDir() string {
	if dir := os.Getenv("GOOGLE_CREDENTIALS_DIR"); dir != "" {
		return dir
	}
	return config.CredentialsDir("google")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
