package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template" // Import the text/template package
	"time"

	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

var currentClientId string

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Send commands to a connected browser client via the agent server",
	Long:  `This command group allows sending various JavaScript evaluation commands to a specific client connected to the agent server.`,
}

// sendEvalScript remains the same as the previous correct version
func sendEvalScript(clientIdFromFlag string, scriptToEvaluate string, waitForResult bool) (any, error) {
	clientId := clientIdFromFlag
	if clientId == "" {
		clientId = strings.TrimSpace(os.Getenv("VIBRANT_CLIENT_ID"))
	}
	if clientId == "" {
		return nil, fmt.Errorf("client id not provided. Pass the -i flag or set VIBRANT_CLIENT_ID envvar")
	}

	evalPayload := scriptToEvaluate // map[string]string{ "scriptToEvaluate": scriptToEvaluate, }
	jsonBody := []byte(evalPayload)

	/*
		jsonBody, err := json.Marshal(evalPayload)
		if err != nil {
			return nil, fmt.Errorf("error marshalling eval payload: %w", err)
		}
	*/

	endpointURL := fmt.Sprintf("http://localhost:9999/agents/%s/eval", clientId)
	if waitForResult {
		endpointURL += "?wait=true"
	}

	req, err := http.NewRequest("POST", endpointURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating new HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 40 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling eval endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return string(respBody), nil
	}

	if waitForResult {
		if respData, ok := result["response"]; ok {
			return respData, nil
		} else if errData, ok := result["error"]; ok { // Check if server itself sent an error in JSON
			return nil, fmt.Errorf("script evaluation error from server: %v", errData)
		}
		return result, nil
	}
	return result, nil
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
		_, err := sendEvalScript(currentClientId, script, false)
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
		_, err := sendEvalScript(currentClientId, script, false)
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
		_, err = sendEvalScript(currentClientId, script, false)
		if err != nil {
			log.Fatalf("Error sending scrolldelta command: %v", err)
		}
		log.Printf("Scroll-delta (%f) command sent.", dy)
	},
}

// --- Template approach for SET_INPUT_VALUE ---
type setInputValueScriptData struct {
	Selector       string // This will be a JSON-encoded string, e.g., "\"#myInput\""
	Value          string // Also JSON-encoded
	Submit         bool
	SubmitSelector string // Also JSON-encoded
}

const setInputValueScriptTplString = `
    (() => {
      const el = document.querySelector({{.Selector}});
      if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
        el.focus();
        el.value = {{.Value}};
        el.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
        el.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
        let report = { success: true, selector: {{.Selector}}, valueSet: true, submitted: false, submitButtonFound: null };
        
        if ({{.Submit}} && {{.SubmitSelector}} !== "\"\"") { // Check if marshalled SubmitSelector is not an empty JSON string ""
            const submitBtn = document.querySelector({{.SubmitSelector}});
            if (submitBtn instanceof HTMLElement) {
                report.submitButtonFound = true;
                if (submitBtn.hasAttribute('disabled')) submitBtn.removeAttribute("disabled");
                const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window });
                submitBtn.dispatchEvent(clickEvent);
                report.submitted = true;
                console.log('[AgentAction] Set value for ' + {{.Selector}} + ' and also clicked submit button: ' + {{.SubmitSelector}});
            } else {
                console.warn('[AgentAction] Set value for ' + {{.Selector}} + ' but submit button not found for selector: ' + {{.SubmitSelector}});
                report.submitButtonFound = false;
            }
        } else {
            console.log('[AgentAction] Set value for selector: ' + {{.Selector}});
        }
        return report;
      } else {
        console.error('[AgentActionError] Could not find input/textarea for selector: ' + {{.Selector}});
        return { success: false, selector: {{.Selector}}, error: 'Element not found or not an input/textarea' };
      }
    })();
`

var jsSetValueTemplate *template.Template // Parsed template

func buildSetInputValueScriptUsingTemplate(selector string, value string, submit bool, submitSelector string) (string, error) {
	sSelectorBytes, err := json.Marshal(selector)
	if err != nil {
		return "", fmt.Errorf("failed to marshal selector: %w", err)
	}
	sValueBytes, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to marshal value: %w", err)
	}
	sSubmitSelectorBytes, err := json.Marshal(submitSelector)
	if err != nil {
		return "", fmt.Errorf("failed to marshal submitSelector: %w", err)
	}

	data := setInputValueScriptData{
		Selector:       string(sSelectorBytes),
		Value:          string(sValueBytes),
		Submit:         submit,
		SubmitSelector: string(sSubmitSelectorBytes),
	}

	var scriptBuf bytes.Buffer
	if err := jsSetValueTemplate.Execute(&scriptBuf, data); err != nil {
		return "", fmt.Errorf("failed to execute setInputValue template: %w", err)
	}
	return scriptBuf.String(), nil
}

// --- End of Template approach ---

var clientCmdToolCallRespond = &cobra.Command{
	Use:   "respond",
	Short: "Sets value in 'ms-function-call-chunk textarea' and optionally submits.",
	Run: func(cmd *cobra.Command, args []string) {
		fromClipboard, _ := cmd.Flags().GetBool("from-clipboard")
		submitFlag, _ := cmd.Flags().GetBool("submit") // Renamed to avoid conflict
		value, err := tools.GetInputFromUserOrClipboard(fromClipboard)
		if err != nil {
			log.Fatalf("Error getting input: %v", err)
		}

		targetSelector := "ms-function-call-chunk textarea"
		targetSubmitSelector := `ms-function-call-chunk footer button[aria-label='Submit']`
		script, err := buildSetInputValueScriptUsingTemplate(targetSelector, value, submitFlag, targetSubmitSelector)
		if err != nil {
			log.Fatalf("Error building script: %v", err)
		}

		response, err := sendEvalScript(currentClientId, script, true)
		if err != nil {
			log.Fatalf("Error sending respond command: %v", err)
		}
		log.Printf("Respond command sent. Result: %v", response)
	},
}

var clientCmdSendPrompt = &cobra.Command{
	Use:   "send",
	Short: "Sets value in 'ms-prompt-input-wrapper textarea' and optionally submits.",
	Run: func(cmd *cobra.Command, args []string) {
		fromClipboard, _ := cmd.Flags().GetBool("from-clipboard")
		submitFlag, _ := cmd.Flags().GetBool("submit") // Renamed
		value, err := tools.GetInputFromUserOrClipboard(fromClipboard)
		if err != nil {
			log.Fatalf("Error getting input: %v", err)
		}

		targetSelector := "ms-prompt-input-wrapper textarea"
		targetSubmitSelector := `ms-prompt-input-wrapper button[aria-label='Run']`
		script, err := buildSetInputValueScriptUsingTemplate(targetSelector, value, submitFlag, targetSubmitSelector)
		if err != nil {
			log.Fatalf("Error building script: %v", err)
		}

		response, err := sendEvalScript(currentClientId, script, true)
		if err != nil {
			log.Fatalf("Error sending prompt command: %v", err)
		}
		log.Printf("Send prompt command sent. Result: %v", response)
	},
}

var clientCmdGetTitle = &cobra.Command{
	Use:   "gettitle",
	Short: "Gets the title of the inspected page",
	Run: func(cmd *cobra.Command, args []string) {
		script := "document.title;"
		response, err := sendEvalScript(currentClientId, script, true)
		if err != nil {
			log.Fatalf("Error sending gettitle command: %v", err)
		}
		log.Printf("Page title: %v", response)
	},
}

func init() {
	// Parse the template for SET_INPUT_VALUE once.
	tpl, err := template.New("setInputValue").Parse(setInputValueScriptTplString)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse internal setInputValueScriptTplString: %v", err))
	}
	jsSetValueTemplate = tpl

	AddCommand(clientCmd)

	clientCmd.AddCommand(clientCmdScrollToTop)
	clientCmd.AddCommand(clientCmdScrollToBottom)
	clientCmd.AddCommand(clientCmdScrollDelta)
	clientCmd.AddCommand(clientCmdToolCallRespond)
	clientCmd.AddCommand(clientCmdSendPrompt)
	clientCmd.AddCommand(clientCmdGetTitle)

	clientCmdToolCallRespond.Flags().BoolP("submit", "s", true, "Induce a 'submit' after the value is set (default true)")

	clientCmdSendPrompt.Flags().BoolP("submit", "s", true, "Induce a 'submit' after the value is set (default true)")
}
