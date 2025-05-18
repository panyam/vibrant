package main

import (
	"log"
	"net/http"

	"github.com/panyam/vibrant/web"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Starts the vibrant agent for quick testing",
	Long:  "Starts the vibrant agent which can serve a frontend only site powered by dynamic templates for testing",
	// Args:  cobra.ExactArgs(3), // metric_type, system_name, analysis_name
	Run: func(cmd *cobra.Command, args []string) {
		// Get the ServeMux from the web package
		agentsMux := web.NewServeMux()

		// Register the agentsMux under the /agents/ prefix on the DefaultServeMux
		// http.DefaultServeMux will be used by b.Serve if its first arg is nil
		http.Handle("/agents/", http.StripPrefix("/agents", agentsMux))
		log.Println("Registered agents API at /agents/")
		if err := http.ListenAndServe(":9999", nil); err != nil {
			panic(err)
		}
	},
}

func init() {
	// Add 'plot' as a subcommand of 'visualize', or directly if 'visualize' is just a namespace.
	// If we had a visualizeCmd:
	// visualizeCmd.AddCommand(agentCmd)
	// Else, add directly:
	AddCommand(agentCmd)

	agentCmd.Flags().StringP("port", "p", "9999", "Port where the agent runs.")
	/*
		agentCmd.MarkFlagRequired("output") // Usually want an output file for plots
		agentCmd.Flags().String("results", "", "Path to a JSON results file (from 'sdl run --json-results')")
		agentCmd.Flags().String("title", "", "Title for the plot")
		agentCmd.Flags().String("x-label", "", "Label for the X-axis")
		agentCmd.Flags().String("y-label", "", "Label for the Y-axis")
		agentCmd.Flags().String("percentiles", "0.5,0.9,0.99", "Comma-separated percentiles to mark on CDF (e.g., '0.5,0.99')")
	*/
}
