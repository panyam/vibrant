package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// Global persistent flags - these will be accessible by all commands.
// Variables are typically defined at the package level if they are to be truly global
// and modified by PersistentPreRun or flag parsing.
const DEFAULT_VIBRANT_PORT = "9999"
const DEFAULT_VIBRANT_HOST = "localhost:9999"

var rootCurrentClientId string
var rootVibrantHost string
var rootFromClipboard bool

// var dslFilePath string // This was from your original root.go, kept for context

var rootCmd = &cobra.Command{
	Use:   "vibrant",
	Short: "Vibrant is a toolchain for working with AI Agents",
	Long:  `Vibrant is a toolchain for working with AI Agents`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// This function runs before any command's Run function.
		// It's a good place to handle logic that applies to all commands,
		// like checking for required persistent flags or loading global config.

		// If the --client-id flag was not used, check the environment variable.
		// Cobra normally handles default values from flags, but for env var precedence
		// before a *required* flag (if it were required), this is one way.
		if !cmd.Flags().Changed("client-id") { // Check if the flag was explicitly set
			envClientId := os.Getenv("VIBRANT_CLIENT_ID")
			if envClientId != "" {
				rootCurrentClientId = envClientId
			}
		}

		if !cmd.Flags().Changed("host") {
			envVibrantHost := os.Getenv("VIBRANT_HOST")
			if envVibrantHost != "" {
				rootVibrantHost = envVibrantHost
			} else {
				log.Println("Using default host: ", DEFAULT_VIBRANT_HOST)
				rootVibrantHost = DEFAULT_VIBRANT_HOST
			}
		}
		// If after checking flag and env, it's still empty, some commands might error out later.
		// rootFromClipboard is handled directly by its flag.
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Define persistent flags on rootCmd
	// The default value for client-id will be "" if VIBRANT_CLIENT_ID is not set.
	// Commands themselves will need to check if rootCurrentClientId is populated.
	rootCmd.PersistentFlags().StringVarP(&rootCurrentClientId, "client-id", "i", os.Getenv("VIBRANT_CLIENT_ID"), "ID of the client. Default from VIBRANT_CLIENT_ID env var if set.")
	rootCmd.PersistentFlags().StringVarP(&rootVibrantHost, "host", "", os.Getenv("VIBRANT_HOST"), fmt.Sprintf("Host to connect our client to.  Default from VIBRANT_CLIENT_ID env var if set otherwise %s.", DEFAULT_VIBRANT_HOST))
	rootCmd.PersistentFlags().BoolVarP(&rootFromClipboard, "from-clipboard", "c", false, "Read input from clipboard instead of from stdin (where applicable).")

	// rootCmd.PersistentFlags().StringVarP(&dslFilePath, "file", "f", "", "Path to the DSL file (required by many commands)")
}

func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}
