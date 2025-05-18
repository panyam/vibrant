package main

import (
	// "github.com/panyam/sdl/decl"
	// "gonum.org/v1/plot" // For actual plotting
	// "gonum.org/v1/plot/plotter"
	// "gonum.org/v1/plot/vg"
	"log"

	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools <subcommand>",
	Short: "Command group to work with and try out tools",
	Long:  `Tools group of commands lets you list tools, try out tools, query them and even configure and inspect them`,
	Run: func(cmd *cobra.Command, args []string) {
		tools.PrintTools()
	},
}

var toolsJsonCmd = &cobra.Command{
	Use:   "json",
	Short: "Prints out all the tools in json format to be uploaded into the AI UX",
	Long:  `Prints out all the tools in json format to be uploaded into the AI UX`,
	// Args:  cobra.ExactArgs(3), // metric_type, system_name, analysis_name
	Run: func(cmd *cobra.Command, args []string) {
		tools.ToolsJson()
	},
}

var runToolCmd = &cobra.Command{
	Use:   "run [TOOLNAME]",
	Short: "Runs a specific tool",
	Long:  "Runs a specific tool",
	// Args:  cobra.ExactArgs(3), // metric_type, system_name, analysis_name
	Args: cobra.MinimumNArgs(1), // metric_type, system_name, analysis_name
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Args: ", args)
		tools.RunTool(args[0], args[1:])
	},
}

func init() {
	// Add 'plot' as a subcommand of 'visualize', or directly if 'visualize' is just a namespace.
	// If we had a visualizeCmd:
	// visualizeCmd.AddCommand(toolsCmd)
	// Else, add directly:
	AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsJsonCmd)
	toolsCmd.AddCommand(runToolCmd)

	/*
		toolsCmd.Flags().StringP("output", "o", "", "Output file path for the plot (e.g., plot.png)")
		toolsCmd.MarkFlagRequired("output") // Usually want an output file for plots
		toolsCmd.Flags().String("results", "", "Path to a JSON results file (from 'sdl run --json-results')")
		toolsCmd.Flags().String("title", "", "Title for the plot")
		toolsCmd.Flags().String("x-label", "", "Label for the X-axis")
		toolsCmd.Flags().String("y-label", "", "Label for the Y-axis")
		toolsCmd.Flags().String("percentiles", "0.5,0.9,0.99", "Comma-separated percentiles to mark on CDF (e.g., '0.5,0.99')")
	*/
}
