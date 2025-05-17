package main

import (
	"os"

	gotl "github.com/panyam/goutils/template"
	tu "github.com/panyam/templar/utils"
)

func main() {
	nargs := len(os.Args)
	if nargs > 1 && os.Args[1] == "tools" {
		if nargs > 2 {
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
