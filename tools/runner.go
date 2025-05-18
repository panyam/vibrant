package tools

import (
	"encoding/json"
	"fmt"
	"log"

	"golang.design/x/clipboard"
)

var tools map[string]Tool

func init() {
	tools = map[string]Tool{
		"read_file":       &ReadFile{BaseFileTool{ProjectRoot: "./"}},
		"list_files":      &ListFiles{BaseFileTool{ProjectRoot: "./"}},
		"create_file":     &CreateFile{BaseFileTool{ProjectRoot: "./"}},
		"apply_file_diff": &ApplyFileDiff{BaseFileTool{ProjectRoot: "./"}},
	}

	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
}

func RunTool(fromClipboard bool, name string, args []string) {
	input, err := GetInputFromUserOrClipboard(fromClipboard)
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
		if val, ok := result.(string); ok {
			clipboard.Write(clipboard.FmtText, []byte(val))
		}
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
