package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	ghttp "github.com/panyam/goutils/http"
	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

var currentClientId string

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Send messages to clients",
	Long:  `A command group for messaging clients`,
}

func sendCommand(cmd string, payload map[string]any) any {
	client := currentClientId
	if client == "" {
		client = strings.TrimSpace(os.Getenv("VIBRANT_CLIENT_ID"))
	}
	if client == "" {
		panic("client id not provided.  Pass the -i flag or set VIBRANT_CLIENT_ID envvar")
	}
	req, err := ghttp.NewJsonRequest("POST", fmt.Sprintf("http://localhost:9999/agents/%s/%s", client, cmd), payload)
	if err != nil {
		panic(err)
	}
	response, err := ghttp.Call(req, nil)
	if err != nil {
		panic(err)
	}
	log.Println("Resp: ", response)
	return response
}

var clientCmdScrollToTop = &cobra.Command{
	Use:   "scrolltop",
	Short: "Scrolls to the top",
	Long:  "Scrolls to the top",
	Run: func(cmd *cobra.Command, args []string) {
		sendCommand("SCROLL_TO_TOP", nil)
	},
}

var clientCmdScrollToBottom = &cobra.Command{
	Use:   "scrollbottom",
	Short: "Scrolls to the bottom",
	Long:  "Scrolls to the bottom",
	Run: func(cmd *cobra.Command, args []string) {
		sendCommand("SCROLL_TO_BOTTOM", nil)
	},
}

var clientCmdScrollDelta = &cobra.Command{
	Use:   "scrolldelta",
	Short: "Scrolls to the delta",
	Long:  "Scrolls to the delta",
	Run: func(cmd *cobra.Command, args []string) {
		dy, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			panic(err)
		}
		log.Println("Delta: ", dy)
		sendCommand("SCROLL_DELTA", map[string]any{"deltaY": dy})
	},
}

var clientCmdToolCallRespond = &cobra.Command{
	Use:   "respond",
	Short: "Responds to a toolcalls",
	Long:  "Responds to a toolcalls",
	Run: func(cmd *cobra.Command, args []string) {
		fromClipboard, _ := cmd.Flags().GetBool("from-clipboard")
		value, err := tools.GetInputFromUserOrClipboard(fromClipboard)
		if err != nil {
			panic(err)
		}
		sendCommand("SET_INPUT_VALUE", map[string]any{
			"selector": "ms-function-call-chunk textarea",
			"value":    value,
		})
	},
}

func init() {
	// Add 'agents' command to root
	AddCommand(clientCmd)

	// Add 'serve' as a subcommand of 'agents'
	clientCmd.AddCommand(clientCmdScrollToTop)
	clientCmd.AddCommand(clientCmdScrollToBottom)
	clientCmd.AddCommand(clientCmdToolCallRespond)

	clientCmdToolCallRespond.Flags().BoolP("from-clipboard", "c", false, "Read input from clipboard instead of from stdin")

	clientCmd.PersistentFlags().StringVarP(&currentClientId, "client-id", "i", "", "ID of the client to default to.  If not provided then the environment VIBRANT_CLIENT_ID is used.")
}
