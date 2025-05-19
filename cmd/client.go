package main

import (
	"fmt"
	"log" // Keep for os.Getenv as a fallback if rootCurrentClientId isn't populated by PersistentPreRun
	"strconv"

	"github.com/spf13/cobra"
)

// Note: currentClientId is now rootCurrentClientId from cmd/root.go (same package 'main')

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Send commands to a connected browser client via the agent server",
	Long:  `This command group allows sending various JavaScript evaluation commands to a specific client connected to the agent server.`,
}

var clientCmdScrollToTop = &cobra.Command{
	Use:   "scrolltop",
	Short: "Scrolls the page (or ms-autoscroll-container) to the top",
	Run: func(cmd *cobra.Command, args []string) {
		script := `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || window;
          scroller.scrollTo(0, 0);
          return 'Scrolled to top';
        })();`
		_, err := sendEvalScript(script, false)
		if err != nil {
			log.Fatalf("Error sending scrolltop command: %v", err)
		}
		log.Println("Scroll-to-top command sent.")
	},
}

var clientCmdScrollToBottom = &cobra.Command{
	Use:   "scrollbottom",
	Short: "Scrolls the page (or ms-autoscroll-container) to the bottom",
	Run: func(cmd *cobra.Command, args []string) {
		script := `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || document.body;
          let target = scroller === window ? document.body.scrollHeight : scroller.scrollHeight;
          (scroller === window ? window : scroller).scrollTo(0, target);
          return 'Scrolled to bottom';
        })();`
		_, err := sendEvalScript(script, false)
		if err != nil {
			log.Fatalf("Error sending scrollbottom command: %v", err)
		}
		log.Println("Scroll-to-bottom command sent.")
	},
}

var clientCmdScrollDelta = &cobra.Command{
	Use:   "scrolldelta [deltaY]",
	Short: "Scrolls by a deltaY value. Use negative for up.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dy, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			log.Fatalf("Invalid deltaY value: %v", err)
		}
		script := fmt.Sprintf(`
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || window;
          (scroller === window ? window : scroller).scrollBy(0, %f);
          return 'Scrolled by deltaY: %f';
        })();`, dy, dy)
		_, err = sendEvalScript(script, false)
		if err != nil {
			log.Fatalf("Error sending scrolldelta command: %v", err)
		}
		log.Printf("Scroll-delta (%f) command sent.", dy)
	},
}

var clientCmdGetTitle = &cobra.Command{
	Use:   "gettitle",
	Short: "Gets the title of the inspected page",
	Run: func(cmd *cobra.Command, args []string) {
		script := "document.title;"
		response, err := sendEvalScript(script, true)
		if err != nil {
			log.Fatalf("Error sending gettitle command: %v", err)
		}
		log.Printf("Page title: %v", response)
	},
}

func init() {

	AddCommand(clientCmd)

	clientCmd.AddCommand(clientCmdScrollToTop)
	clientCmd.AddCommand(clientCmdScrollToBottom)
	clientCmd.AddCommand(clientCmdScrollDelta)
	clientCmd.AddCommand(clientCmdGetTitle)
}
