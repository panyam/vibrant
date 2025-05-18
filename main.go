package main

import (
	"encoding/json"
	"fmt"
	"os"

	gotl "github.com/panyam/goutils/template"
	tu "github.com/panyam/templar/utils"
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

	// Start the server
	b := tu.BasicServer{
		FuncMaps: []map[string]any{
			gotl.DefaultFuncMap(),
		},
	}
	b.Serve(nil, ":7777")
}

var tools map[string]Tool

func init() {
	tools = map[string]Tool{
		"read_file":  &ReadFile{ProjectRoot: "./"},
		"list_files": &ListFiles{ProjectRoot: "./"},
		"edit_file":  &EditFile{ProjectRoot: "./"},
	}
}

func RunTool(name string, args []string) {
	tools[name].Run(args)
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
