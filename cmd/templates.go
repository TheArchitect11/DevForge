package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/registry"
	"github.com/chinmay/devforge/internal/ux"
)

const defaultRegistryURL = "https://thearchitect11.github.io/devforge-registry/templates.json"

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Browse the starter template registry",
	Long: `List, search, and inspect starter templates from the DevForge template registry.

  devforge templates list               # all templates
  devforge templates search <keyword>   # filter by name, description, or tag
  devforge templates use <name>         # detailed info + usage hint`,
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	RunE:  runTemplatesList,
}

var templatesSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search templates by keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplatesSearch,
}

var templatesUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Show details for a template and how to use it",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplatesUse,
}

var refresh bool

func init() {
	templatesCmd.PersistentFlags().BoolVar(&refresh, "refresh", false, "bypass the local cache and force a registry fetch")
	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesSearchCmd)
	templatesCmd.AddCommand(templatesUseCmd)
	rootCmd.AddCommand(templatesCmd)
}

// getRegistryClient creates a logger and registry client, returning both
// so the caller can defer log.Close().
func getRegistryClient() (*registry.Client, *logger.Logger, error) {
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return nil, nil, fmt.Errorf("logger initialization failed: %w", err)
	}
	return registry.NewClient(defaultRegistryURL, log), log, nil
}

func runTemplatesList(_ *cobra.Command, _ []string) error {
	client, log, err := getRegistryClient()
	if err != nil {
		return err
	}
	defer log.Close()

	spin := ux.NewSpinner("Fetching template registry")
	reg, err := client.Fetch(refresh)
	if err != nil {
		spin.Fail("Could not reach registry")
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}
	spin.Stop("")

	templates := reg.ValidTemplates()
	if len(templates) == 0 {
		ux.Warning("The registry contains no templates.")
		return nil
	}

	ux.Header(fmt.Sprintf("Available Templates (%d)", len(templates)))
	fmt.Printf("  %-22s %-42s %s\n", "Name", "Description", "Tags")
	fmt.Printf("  %-22s %-42s %s\n", "────", "───────────", "────")

	for _, t := range templates {
		desc := t.Description
		if len(desc) > 40 {
			desc = desc[:37] + "…"
		}
		tags := strings.Join(t.Tags, ", ")
		fmt.Printf("  %-22s %-42s %s\n", t.Name, desc, tags)
	}

	fmt.Println()
	fmt.Println("  Run: devforge templates use <name>   for details")
	fmt.Println("       devforge init <project> --template <name>   to scaffold")
	fmt.Println()
	return nil
}

func runTemplatesSearch(_ *cobra.Command, args []string) error {
	keyword := args[0]

	client, log, err := getRegistryClient()
	if err != nil {
		return err
	}
	defer log.Close()

	spin := ux.NewSpinner(fmt.Sprintf("Searching registry for %q", keyword))
	reg, err := client.Fetch(refresh)
	if err != nil {
		spin.Fail("Could not reach registry")
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}
	spin.Stop("")

	results := client.Search(reg, keyword)
	if len(results) == 0 {
		ux.Warning("No templates matched %q.", keyword)
		fmt.Println("  Try: devforge templates list")
		return nil
	}

	ux.Header(fmt.Sprintf("Search results for %q (%d)", keyword, len(results)))
	for _, t := range results {
		fmt.Printf("  %s %s\n", ux.Arrow, t.Name)
		fmt.Printf("    %s\n", t.Description)
		fmt.Printf("    %s\n", t.URL)
		if len(t.Tags) > 0 {
			fmt.Printf("    tags: %s\n", strings.Join(t.Tags, ", "))
		}
		fmt.Println()
	}
	fmt.Printf("  Use: devforge init <project> --template <name>\n\n")
	return nil
}

func runTemplatesUse(_ *cobra.Command, args []string) error {
	name := args[0]

	client, log, err := getRegistryClient()
	if err != nil {
		return err
	}
	defer log.Close()

	spin := ux.NewSpinner(fmt.Sprintf("Looking up %q", name))
	reg, err := client.Fetch(refresh)
	if err != nil {
		spin.Fail("Could not reach registry")
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}
	spin.Stop("")

	for _, t := range reg.ValidTemplates() {
		if strings.EqualFold(t.Name, name) { // case-insensitive match
			ux.Header(fmt.Sprintf("Template: %s", t.Name))
			fmt.Printf("  Description : %s\n", t.Description)
			fmt.Printf("  Author      : %s\n", t.Author)
			fmt.Printf("  Version     : %s\n", t.Version)
			fmt.Printf("  URL         : %s\n", t.URL)
			if len(t.Tags) > 0 {
				fmt.Printf("  Tags        : %s\n", strings.Join(t.Tags, ", "))
			}
			fmt.Println()
			ux.InfoMsg("To use this template:")
			fmt.Printf("    devforge init my-project --template %s\n\n", t.Name)
			return nil
		}
	}

	ux.Error(fmt.Errorf("template %q not found", name))
	fmt.Println("  Run 'devforge templates list' to see all available templates.")
	return nil
}

var templatesClearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Remove the local template registry cache",
	Long: `Delete the locally cached copy of the template registry
(~/.devforge/cache/templates.json).

The next 'devforge templates list' or 'devforge init' call will fetch
fresh data from the remote registry.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		client, log, err := getRegistryClient()
		if err != nil {
			return err
		}
		defer log.Close()
		if err := client.ClearCache(); err != nil {
			return err
		}
		ux.Success("Template registry cache cleared.")
		return nil
	},
}

func init() {
	templatesCmd.AddCommand(templatesClearCacheCmd)
}
