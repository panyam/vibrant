package tools

import (
	"os"
	"path/filepath"
)

type ListFiles struct {
	BaseFileTool
}

func (r *ListFiles) Name() string {
	return "list_files"
}

func (r *ListFiles) Description() string {
	return `
	Lists the files in a given folder provided by the 'path' parameter.   The 'recurse' parameter enables recursive file listing.

	Returns an error or a list of entries.  Each entry is a file or a folder.   Folder entries could recursively contain other entries.
	`
}

func (r *ListFiles) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the folder to list files in.  Path will be resolved to a relative path in the current project",
			Type:        "string",
		},
		{
			Name:        "recurse",
			Description: "Whether to return files recursively.  This will help with minimizing number of tool calls if the folder content changes (file creations and deletions) are minimal",
			Type:        "boolean",
		},
	}
}

func (r *ListFiles) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "entries",
			Description: "List of entries.  Each entry can be a folder or a file.  If the entry is a folder, then it may recursively contain entries - if the recurse parameter was set to true",
			Type:        "object",
		},
	}
}

func (r *ListFiles) Run(args map[string]any) (any, error) {
	path := args["path"].(string)
	dir, err := filepath.Abs(filepath.Join(r.ProjectRoot, path))
	if err != nil {
		return nil, err
	}

	var entries []any
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		entry := map[string]any{}
		if relPath != "." {
			if info.IsDir() {
				entry = map[string]any{
					"is_folder": true,
				}
			} else {
				entry = map[string]any{
					"path": relPath,
				}
			}
			entries = append(entries, entry)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, err
}
