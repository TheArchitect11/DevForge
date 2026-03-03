package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/registry"
)

const defaultRegistryURL = "https://thearchitect11.github.io/devforge-registry/templates.json"

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage starter templates from the remote registry",
	Long:  `Browse, search, and select starter templates from the DevForge template registry.`,
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
	Short: "Display details for a specific template",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplatesUse,
}
var refresh bool

func init() {
	templatesCmd.PersistentFlags().BoolVar(&refresh, "refresh", false, "force refresh of the template registry cache")

	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesSearchCmd)
	templatesCmd.AddCommand(templatesUseCmd)
	rootCmd.AddCommand(templatesCmd)
}

func getRegistryClient() (*registry.Client, *logger.Logger, error) {
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return nil, nil, fmt.Errorf("logger initialization failed: %w", err)
	}
	client := registry.NewClient(defaultRegistryURL, log)
	return client, log, nil
}

func runTemplatesList(_ *cobra.Command, _ []string) error {
	client, log, err := getRegistryClient()
	if err != nil {
		return err
	}
	defer log.Close()

	reg, err := client.Fetch(refresh)
	if err != nil {
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}

	templates := reg.ValidTemplates()
	if len(templates) == 0 {
		fmt.Println("No templates available.")
		return nil
	}

	fmt.Println()
	fmt.Println("  Available Templates")
	fmt.Println("  ═══════════════════")
	fmt.Println()
	fmt.Printf("  %-20s %-40s %s\n", "Name", "Description", "URL")
	fmt.Printf("  %-20s %-40s %s\n", "────", "───────────", "───")
	for _, t := range templates {
		desc := t.Description
		if len(desc) > 38 {
			desc = desc[:35] + "..."
		}
		fmt.Printf("  %-20s %-40s %s\n", t.Name, desc, t.URL)
	}
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

	reg, err := client.Fetch(refresh)
	if err != nil {
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}

	results := client.Search(reg, keyword)
	if len(results) == 0 {
		fmt.Printf("No templates found matching %q.\n", keyword)
		return nil
	}

	fmt.Printf("\n  Search results for %q:\n\n", keyword)
	for _, t := range results {
		fmt.Printf("  • %s — %s\n    %s\n\n", t.Name, t.Description, t.URL)
	}

	return nil
}

func runTemplatesUse(_ *cobra.Command, args []string) error {
	name := args[0]

	client, log, err := getRegistryClient()
	if err != nil {
		return err
	}
	defer log.Close()

	reg, err := client.Fetch(refresh)
	if err != nil {
		return fmt.Errorf("failed to fetch template registry: %w", err)
	}

	for _, t := range reg.ValidTemplates() {
		if t.Name == name {
			fmt.Println()
			fmt.Printf("  Template: %s\n", t.Name)
			fmt.Printf("  Description: %s\n", t.Description)
			fmt.Printf("  URL: %s\n", t.URL)
			fmt.Printf("  Author: %s\n", t.Author)
			fmt.Printf("  Version: %s\n", t.Version)
			if len(t.Tags) > 0 {
				fmt.Printf("  Tags: %v\n", t.Tags)
			}
			fmt.Println()
			fmt.Printf("  Usage: devforge init my-project --config <config-with-this-template>\n")
			fmt.Println()
			return nil
		}
	}

	return fmt.Errorf("template %q not found in registry", name)
}
