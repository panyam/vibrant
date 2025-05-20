package tools

import (
	"fmt"
	"os"
)

type WriteFile struct {
	BaseFileTool
}

func (r *WriteFile) Name() string {
	return "write_file"
}

func (r *WriteFile) Description() string {
	return `Creates and makes sure a file exists with the given contents.  If a given file already exists, then it is overridden with the new contents.`
}

func (r *WriteFile) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the file to write or overwrite.",
			Type:        "string",
		},
		{
			Name:        "contents",
			Description: "Text contents to set in the file that is writed (or overwritten)",
			Type:        "string",
		},
	}
}

func (r *WriteFile) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "result",
			Description: "Error or Number of bytes written",
			Type:        "object",
		},
	}
}

func (r *WriteFile) Run(args map[string]any) (any, error) {
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
