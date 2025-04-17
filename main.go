package main

import (
	tu "github.com/panyam/templar/utils"
)

func main() {
	b := tu.BasicServer{}
	b.Serve(nil, ":7777")
}
