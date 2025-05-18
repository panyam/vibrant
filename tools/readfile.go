package tools

import "os"

type ReadFile struct {
	BaseFileTool
}

func (r *ReadFile) Name() string {
	return "read_file"
}

func (r *ReadFile) Description() string {
	return `
	Reads a file with the path given in the 'path' parameter.  The path of the file is ALWAYS relative to the project root.

	Returns 2 values as json:

	content: The content bytes of the file
	metadata: Metadata of the file as a json dictionary.
	`
}

func (r *ReadFile) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the file to read contents for.  Path will be resolved to a relative path in the current project",
			Type:        "string",
		},
	}
}

func (r *ReadFile) Returns() []*Parameter {
	return []*Parameter{
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
	}
}

func (r *ReadFile) Run(args map[string]any) (any, error) {
	path := args["path"].(string)
	fullpath, err := r.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullpath)
	if err != nil {
		return nil, err
	}
	return string(content), nil
}
