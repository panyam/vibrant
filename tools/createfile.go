package tools

import (
	"fmt"
	"os"
)

type CreateFile struct {
	BaseFileTool
}

func (r *CreateFile) Name() string {
	return "create_file"
}

func (r *CreateFile) Description() string {
	return `Creates and makes sure a file exists with the given contents.  If a given file already exists, then it is overridden with the new contents.`
}

func (r *CreateFile) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the file to create or overwrite.",
			Type:        "string",
		},
		{
			Name:        "contents",
			Description: "Text contents to set in the file that is created (or overwritten)",
			Type:        "string",
		},
	}
}

func (r *CreateFile) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "result",
			Description: "Error or Number of bytes written",
			Type:        "object",
		},
	}
}

func (r *CreateFile) Run(args map[string]any) (any, error) {
	path := args["path"].(string)
	fullpath, err := r.ResolvePath(path)
	if err != nil {
		return nil, err
	}
	contents := args["contents"].(string)

	// TODO - replace with stat
	_, err = os.ReadFile(fullpath)
	if err != nil {
		if os.IsNotExist(err) {
			return createNewFile(fullpath, contents)
		}
		return "", err
	}

	err = os.WriteFile(fullpath, []byte(contents), 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Written %d bytes", len([]byte(contents))), nil
}
