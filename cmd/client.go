package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os" // Keep for os.Getenv as a fallback if rootCurrentClientId isn't populated by PersistentPreRun
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

// Note: currentClientId is now rootCurrentClientId from cmd/root.go (same package 'main')

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Send commands to a connected browser client via the agent server",
	Long:  `This command group allows sending various JavaScript evaluation commands to a specific client connected to the agent server.`,
}

// sendEvalScript now uses rootCurrentClientId
// It sends the raw scriptToEvaluate string as the POST body.
func sendEvalScript(scriptToEvaluate string, waitForResult bool) (any, error) {
	// rootCurrentClientId should be populated by Cobra's persistent flag handling by now.
	// The PersistentPreRunE in root.go also attempts to set it from ENV if the flag wasn't used.
	clientIdToUse := rootCurrentClientId
	if clientIdToUse == "" { // Fallback if PersistentPreRun didn't set it
		clientIdToUse = strings.TrimSpace(os.Getenv("VIBRANT_CLIENT_ID"))
	}

	if clientIdToUse == "" {
		return nil, fmt.Errorf("client id not provided. Use -i flag or set VIBRANT_CLIENT_ID env var")
	}

	// The /eval endpoint in web/server.go now expects the raw script as the body.
	// It will internally create a requestId.
	// If wait=true, it still expects a JSON response like {"requestId": "...", "response": ...}
	// If wait=false, it responds with {"status": "...", "requestId": "..."}

	endpointURL := fmt.Sprintf("http://localhost:9999/agents/%s/eval", clientIdToUse)
	if waitForResult {
		endpointURL += "?wait=true"
	}

	req, err := http.NewRequest("POST", endpointURL, strings.NewReader(scriptToEvaluate))
	if err != nil {
		return nil, fmt.Errorf("error creating new HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

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
		log.Printf("Could not unmarshal JSON response from /eval: %v. Raw response: %s", err, string(respBody))
		return string(respBody), nil
	}

	if waitForResult {
		if respData, ok := result["response"]; ok {
			return respData, nil
		} else if errData, ok := result["error"]; ok {
			return nil, fmt.Errorf("server indicated script evaluation error: %v", errData)
		}
		log.Printf("Warning: /eval?wait=true response did not contain 'response' or 'error' field: %v", result)
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

type setInputValueScriptData struct {
	Selector       string
	Value          string
	Submit         bool
	SubmitSelector string
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
        
        if ({{.Submit}} && {{.SubmitSelector}} !== "\"\"") { 
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

var jsSetValueTemplate *template.Template

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

var clientCmdToolCallRespond = &cobra.Command{
	Use:   "respond",
	Short: "Sets value in 'ms-function-call-chunk textarea' and optionally submits.",
	Run: func(cmd *cobra.Command, args []string) {
		// fromClipboard is now rootFromClipboard from cmd/root.go
		submitFlag, _ := cmd.Flags().GetBool("submit")
		value, err := tools.GetInputFromUserOrClipboard(rootFromClipboard) // Use global rootFromClipboard
		if err != nil {
			log.Fatalf("Error getting input: %v", err)
		}

		targetSelector := "ms-function-call-chunk textarea"
		targetSubmitSelector := `ms-function-call-chunk footer button[aria-label='Submit']`
		script, err := buildSetInputValueScriptUsingTemplate(targetSelector, value, submitFlag, targetSubmitSelector)
		if err != nil {
			log.Fatalf("Error building script: %v", err)
		}

		response, err := sendEvalScript(script, false)
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
		// fromClipboard is now rootFromClipboard from cmd/root.go
		submitFlag, _ := cmd.Flags().GetBool("submit")
		value, err := tools.GetInputFromUserOrClipboard(rootFromClipboard) // Use global rootFromClipboard
		if err != nil {
			log.Fatalf("Error getting input: %v", err)
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
