package main

import (
	"log"
	"net/http"

	"github.com/panyam/vibrant/web" // Your web package
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage and run agent-related services",
	Long:  `A command group for managing agent services, like the WebSocket server.`,
}

var agentsServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the agent WebSocket server",
	Long:  `Starts an HTTP server that handles WebSocket connections for agents.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		addr := ":" + port

		log.Printf("Starting Agent WebSocket server on %s", addr)

		// Get the ServeMux from the web package.
		// This ServeMux is configured to handle paths like /agents/{connectionName}/subscribe
		agentApiMux := web.NewServeMux()
		http.Handle("/agents/", http.StripPrefix("/agents", agentApiMux))

		// Start the HTTP server with this specific mux.
		// No StripPrefix is needed here because agentApiMux handles the root path
		// from its perspective (e.g., /agents/..., /test_broadcast)
		err := http.ListenAndServe(addr, agentApiMux)
		if err != nil {
			log.Fatalf("Failed to start agent server on %s: %v", addr, err)
		}
	},
}

func init() {
	// Add 'agents' command to root
	AddCommand(agentsCmd)

	// Add 'serve' as a subcommand of 'agents'
	agentsCmd.AddCommand(agentsServeCmd)

	// Add flags for the 'agents serve' command
	agentsServeCmd.Flags().StringP("port", "p", "9999", "Port for the agent WebSocket server.")
}
