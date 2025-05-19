package main

import (
	"log"
	"strings"

	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

func callsCmdSendPrompt() *cobra.Command {
	out := &cobra.Command{
		Use:   "send",
		Short: "Sets value in 'ms-prompt-input-wrapper textarea' and optionally submits.",
		Run: func(cmd *cobra.Command, args []string) {
			// fromClipboard is now rootFromClipboard from cmd/root.go
			submitFlag, _ := cmd.Flags().GetBool("submit")
			var err error
			var value string
			if len(args) > 0 {
				value = strings.TrimSpace(args[0])
			}

			if value == "" {
				value, err = tools.GetInputFromUserOrClipboard(rootFromClipboard, "Your Prompt") // Use global rootFromClipboard
			}
			if err != nil {
				log.Fatalf("Error getting input: %v", err)
			}

			if value == "" {
				return
			}

			targetSelector := "ms-prompt-input-wrapper textarea"
			targetSubmitSelector := `ms-prompt-input-wrapper button[aria-label='Run']`
			script, err := buildSetInputValueScriptUsingTemplate(targetSelector, value, submitFlag, targetSubmitSelector)
			if err != nil {
				log.Fatalf("Error building script: %v", err)
			}

			response, err := sendEvalScript(script, false)
			if err != nil {
				log.Fatalf("Error sending prompt command: %v", err)
			}
			log.Printf("Send prompt command sent. Result: %v", response)
		},
	}
	out.Flags().BoolP("submit", "s", true, "Induce a 'submit' after the value is set (default true)")
	return out
}

func init() {
	AddCommand(callsCmdSendPrompt())
}
