package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

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

	endpointURL := fmt.Sprintf("http://%s/agents/%s/eval", rootVibrantHost, clientIdToUse)
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
								setTimeout(() => {
									// Give a small delay 
									submitBtn.removeAttribute("disabled");
									const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window });
									submitBtn.dispatchEvent(clickEvent);
								}, 100)
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

func init() {
	tpl, err := template.New("setInputValue").Parse(setInputValueScriptTplString)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse internal setInputValueScriptTplString: %v", err))
	}
	jsSetValueTemplate = tpl
}
