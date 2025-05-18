package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http" // Added for http.Handle and http.StripPrefix
	"os"

	gotl "github.com/panyam/goutils/template"
	"github.com/panyam/vibrant/web" // Added for your web package
)

func main() {
	nargs := len(os.Args)
	if nargs > 1 && os.Args[1] == "tools" {
		if nargs > 2 {
			if os.Args[2] == "json" {
				ToolsJson()
				return
			}
			if os.Args[2] == "run" {
				RunTool(os.Args[3], os.Args[4:])
				return
			}
		}
		PrintTools()
		return
	}

	// Get the ServeMux from the web package
	agentsMux := web.NewServeMux()

	// Register the agentsMux under the /agents/ prefix on the DefaultServeMux
	// http.DefaultServeMux will be used by b.Serve if its first arg is nil
	http.Handle("/agents/", http.StripPrefix("/agents", agentsMux))
	log.Println("Registered agents API at /agents/")

	// Start the server
	b := tu.BasicServer{
		FuncMaps: []map[string]any{
			gotl.DefaultFuncMap(),
		},
	}
	b.Serve(nil, ":7777") // Assuming this uses http.DefaultServeMux
}

var tools map[string]Tool

func init() {
	tools = map[string]Tool{
		"read_file":       &ReadFile{BaseFileTool{ProjectRoot: "./"}},
		"list_files":      &ListFiles{BaseFileTool{ProjectRoot: "./"}},
		"create_file":     &CreateFile{BaseFileTool{ProjectRoot: "./"}},
		"apply_file_diff": &ApplyFileDiff{BaseFileTool{ProjectRoot: "./"}},
	}
}

func RunTool(name string, args []string) {
	fmt.Print("\u001b[94mEnter Tool Call Params\u001b[0m: ")
	input, err := getUserMessageTillEOF()
	if err != nil {
		log.Println("Error reading input: ", err)
		return
	}

	// log.Println("Got Input: ", input)

	var params map[string]any
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		log.Println("Unable to parse tool call json: ", err)
		return
	}

	// log.Println("Params: ", params)
	result, err := tools[name].Run(params)
	if err != nil {
		log.Printf("error: %v", err)
	} else {
		fmt.Println("\nTOOL CALLED SUCCESSFULLY.  Result: ")
		fmt.Println(result)
	}
}

func ToolsJson() {
	var out []any
	for _, tool := range tools {
		props := map[string]any{}
		for _, param := range tool.Parameters() {
			props[param.Name] = param.Json()
		}
		out = append(out, map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  map[string]any{"type": "object", "properties": props},
		})
	}

	b, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	fmt.Println("JSON: ")
	fmt.Println(string(b))
}

func PrintTools() {
	fmt.Println("Tools:")
	for name, tool := range tools {
		fmt.Printf("	%s: \n", name)
		fmt.Println(tool.Description())
		fmt.Println("	Parameters:")
		for _, param := range tool.Parameters() {
			fmt.Printf("		%s (%s) - %s\n", param.Name, param.Type, param.Description)
		}
		fmt.Println("	Returns:")
		for _, param := range tool.Returns() {
			fmt.Printf("		%s (%s) - %s\n", param.Name, param.Type, param.Description)
		}
		fmt.Println("===============================")
	}
}

// Dummy implementation for getUserMessageTillEOF if it's not already defined
// in your actual BaseFileTool or elsewhere. Replace with your actual implementation.
// If it's part of an unshared package, this might cause a compile error later,
// but it's needed for the file to be syntactically plausible based on its usage.
func getUserMessageTillEOF() (string, error) {
	// This is a placeholder.
	// In your actual environment, this function reads user input.
	// For the purpose of making this file content valid for the tool:
	// return "", nil
	panic("getUserMessageTillEOF needs to be implemented or available in the build context")
}
