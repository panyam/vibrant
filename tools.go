package main

type Parameter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"input_params"`
	Returns     []Parameter `json:"output_param"`
}

var ReadFile = Tool{
	Name: "read_file",
	Description: `
	Reads a file with the path given in the 'path' parameter.  The path of the file is ALWAYS relative to the project root.

	Returns 2 values as json:

	content: The content bytes of the file
	metadata: Metadata of the file as a json dictionary.
	`,
	Parameters: []Parameter{
		{
			Name:        "path",
			Description: "Path of the file to read contents for.  Path will be resolved to a relative path in the current project",
			Type:        "string",
		},
	},
	Returns: []Parameter{
		{
			Name:        "contents",
			Description: "Contents of the file if it exists.",
			Type:        "string",
		},
		{
			Name:        "error",
			Description: "If the file does not exist or cannot be read then an error is returned",
			Type:        "string",
		},
	},
}

func RunTool(name string, args []string) {
}

func PrintTools() {
}
