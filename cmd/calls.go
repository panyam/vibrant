package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log" // Keep for os.Getenv as a fallback if rootCurrentClientId isn't populated by PersistentPreRun
	"strconv"
	"text/template"

	"github.com/panyam/vibrant/tools"
	"github.com/spf13/cobra"
)

// Note: currentClientId is now rootCurrentClientId from cmd/root.go (same package 'main')

var callsCmd = &cobra.Command{
	Use:   "calls",
	Short: "Manages pending tool calls",
	Long:  `Lists, repsonds to and tracks call results on the ai studio page`,
}

func listCalls() []map[string]any {
	const script = `
    (() => {
			const retval = [];
      const el = document.querySelectorAll("ms-function-call-chunk");
			el.forEach((node, index) => {
					const nameNode = node.querySelector("mat-panel-title span[class='name']")
					const contentNode = node.querySelector("ms-code-block pre code")
					const submitButton = node.querySelector("footer button[aria-label='Submit']")
					if (nameNode && contentNode && submitButton) {
							retval.push({
									"name": nameNode.innerText,
									"payload": contentNode.innerText,
							})
					}
			})
			console.log("Found Tool Calls: ", retval)
			return retval;
    })();
`

	response, err := sendEvalScript(script, true)
	if err != nil {
		log.Fatalf("Error sending respond command: %v", err)
	}

	var out []map[string]any

	if calllist, ok := response.([]any); ok {
		for idx, call := range calllist {
			if callinfo, ok := call.(map[string]any); ok {
				out = append(out, callinfo)
				var payload map[string]any
				if callinfo["payload"] != nil {
					if err := json.Unmarshal([]byte(callinfo["payload"].(string)), &payload); err != nil {
						log.Println("Error extracting payload params: ", idx, callinfo["payload"])
					} else {
						callinfo["payload"] = payload
					}
				}
			}
		}
	} else {
		log.Printf("Invalid call list: %v", response)
	}
	return out
}

func callsCmdListCalls() *cobra.Command {
	out := &cobra.Command{
		Use:   "list",
		Short: "Lists the pending calls and their details.",
		Run: func(cmd *cobra.Command, args []string) {
			calllist := listCalls()
			if len(calllist) == 0 {
				log.Printf("No pending calls")
			} else {
				for idx, callinfo := range calllist {
					fmt.Printf("============  Call %d - %s ============\n", idx, callinfo["name"])
					if callinfo["payload"] != nil {
						// convert to json
						fmt.Println("Parameters: ")
						payload := callinfo["payload"].(map[string]any)
						i := 1
						for k, v := range payload {
							fmt.Printf("#### %d - %s = %v\n\n", i, k, v)
							i += 1
						}
					}
				}
			}
		},
	}
	return out
}

func callsCmdToolCallRespond() *cobra.Command {
	out := &cobra.Command{
		Use:   "respond CALLNUMBER",
		Short: "Sets value in 'ms-function-call-chunk textarea' and optionally submits.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			callIndex, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				log.Println("invalid index: ", args[0])
				return
			}

			// TODO - keep this in a cache somewhere instead of fetching each time
			allcalls := listCalls()
			if callIndex >= int64(len(allcalls)) || callIndex < 0 {
				log.Printf("call index must be between 0 and %d\n", len(allcalls)-1)
				return
			}

			callinfo := allcalls[callIndex]
			toolname := callinfo["name"].(string)
			toolparams := callinfo["payload"].(map[string]any)
			result, err := tools.RunTool(false, toolname, toolparams)
			if err != nil {
				// TODO - Should we send the error back?
				log.Printf("error running tool '%s': %v", toolname, err)
				return
			}

			// We have the result so now send it back!

			submitFlag, _ := cmd.Flags().GetBool("submit")
			value := result.(string)
			valueEscaped, err := json.Marshal(value)
			if err != nil {
				panic(err)
			}

			const scriptTemplate = `
        (() => {
          const retval = [];
          const el = document.querySelectorAll("ms-function-call-chunk");
          let index = 0;
          el.forEach((node) => {
              const nameNode = node.querySelector("mat-panel-title span[class='name']")
              const contentNode = node.querySelector("ms-code-block pre code")
              const submitButton = node.querySelector("footer button[aria-label='Submit']")
							console.log("Node: ", node)
              if (nameNode && contentNode && submitButton) {
                  if (index == {{.callIndex}}) {
                      const textarea = node.querySelector("textarea")
                      if (textarea) {
                          // set it
													// textarea.classList.remove("ng-untouched")
													// textarea.classList.remove("ng-pristine")
													// textarea.classList.add("ng-dirty")
													textarea.focus();
													textarea.value = {{.value}};
													textarea.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
													textarea.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));

													submitButton.removeAttribute("disabled");
													// submitButton.classList.remove("invalid");
													// submitButton.classList.remove("mat-mdc-button-disabled");

                          console.log("Submit Flag: ", {{.submit}})
                          {{ if .submit }}
                          setTimeout(() => {
															const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window });
                              submitButton.dispatchEvent(clickEvent);
                          }, 500)
                          {{ end }}
                      }
                  }
                  index += 1;
              }
          })
          console.log("Found Tool Calls: ", retval)
          return retval;
        })();
    `
			tpl, err := template.New("setCallResult").Parse(scriptTemplate)
			if err != nil {
				panic(fmt.Sprintf("Failed to parse internal setCallResult: %v", err))
			}
			var scriptBuf bytes.Buffer
			if err := tpl.Execute(&scriptBuf, map[string]any{
				"callIndex": callIndex,
				"value":     string(valueEscaped),
				"submit":    submitFlag,
			}); err != nil {
				log.Fatalf("failed to execute setInputValue template: %v", err)
			}
			script := scriptBuf.String()

			response, err := sendEvalScript(script, false)
			if err != nil {
				log.Fatalf("Error sending respond command: %v", err)
			}
			log.Printf("Respond command sent. Result: %v", response)
		},
	}
	out.Flags().BoolP("submit", "s", false, "Induce a 'submit' after the value is set (default true)")
	return out
}

func init() {
	AddCommand(callsCmd)

	callsCmd.AddCommand(callsCmdListCalls())
	callsCmd.AddCommand(callsCmdToolCallRespond())
}
