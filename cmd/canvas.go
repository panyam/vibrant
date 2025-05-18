package main

import (
	gotl "github.com/panyam/goutils/template"
	tu "github.com/panyam/templar/utils"
	"github.com/spf13/cobra"
)

var canvasCmd = &cobra.Command{
	Use:   "canvas",
	Short: "Starts the vibrant canvas for quick testing",
	Long:  "Starts the vibrant canvas which can serve a frontend only site powered by dynamic templates for testing",
	// Args:  cobra.ExactArgs(3), // metric_type, system_name, analysis_name
	Run: func(cmd *cobra.Command, args []string) {
		// Start the server
		b := tu.BasicServer{
			FuncMaps: []map[string]any{
				gotl.DefaultFuncMap(),
			},
		}
		b.Serve(nil, ":7777") // Assuming this uses http.DefaultServeMux
	},
}

func init() {
	// Add 'plot' as a subcommand of 'visualize', or directly if 'visualize' is just a namespace.
	// If we had a visualizeCmd:
	// visualizeCmd.AddCommand(canvasCmd)
	// Else, add directly:
	AddCommand(canvasCmd)

	canvasCmd.Flags().StringP("port", "p", "7777", "Port where canvas runs.")
	/*
		canvasCmd.MarkFlagRequired("output") // Usually want an output file for plots
		canvasCmd.Flags().String("results", "", "Path to a JSON results file (from 'sdl run --json-results')")
		canvasCmd.Flags().String("title", "", "Title for the plot")
		canvasCmd.Flags().String("x-label", "", "Label for the X-axis")
		canvasCmd.Flags().String("y-label", "", "Label for the Y-axis")
		canvasCmd.Flags().String("percentiles", "0.5,0.9,0.99", "Comma-separated percentiles to mark on CDF (e.g., '0.5,0.99')")
	*/
}
