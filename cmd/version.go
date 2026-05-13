package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/ux"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print DevForge version and build information",
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(_ *cobra.Command, _ []string) {
	ux.Banner(fmt.Sprintf("v%s", Version))
	fmt.Printf("  Binary  : devforge v%s\n", Version)
	fmt.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Go      : %s\n", runtime.Version())
	fmt.Println()
	fmt.Println("  Run 'devforge update' to check for a newer release.")
	fmt.Println()
}
