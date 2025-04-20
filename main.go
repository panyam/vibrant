package main

import (
	gotl "github.com/panyam/goutils/template"
	tu "github.com/panyam/templar/utils"
)

func main() {
	b := tu.BasicServer{
		FuncMaps: []map[string]any{
			gotl.DefaultFuncMap(),
		},
	}
	b.Serve(nil, ":7777")
}
