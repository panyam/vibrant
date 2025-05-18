package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Parameter struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

func (p *Parameter) Json() any {
	return map[string]any{
		"description": p.Description,
		"type":        p.Type,
	}
}

type Tool interface {
	Name() string
	Description() string
	Parameters() []*Parameter
	Returns() []*Parameter
	Run(args []string) (any, error)
}

type ReadFile struct {
	ProjectRoot string
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

func (r *ReadFile) Run(args []string) (any, error) {
	path := args[0]
	fullpath, err := filepath.Abs(filepath.Join(r.ProjectRoot, path))
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullpath)
	if err != nil {
		return nil, err
	}
	return string(content), nil
}

type ListFiles struct {
	ProjectRoot string
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

func (r *ListFiles) Run(args []string) (any, error) {
	path := args[0]
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

type EditFile struct {
	ProjectRoot string
}

func (r *EditFile) Name() string {
	return "edit_file"
}

func (r *EditFile) Description() string {
	return `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

If the file specified with path doesn't exist, it will be created. `
}

func (r *EditFile) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the folder to list files in.  Path will be resolved to a relative path in the current project",
			Type:        "string",
		},
		{
			Name:        "old_str",
			Description: "Text to search for - must match exactly and must only have one match exactly",
			Type:        "string",
		},
		{
			Name:        "new_str",
			Description: "Text to replace old_str with",
			Type:        "string",
		},
	}
}

func (r *EditFile) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "entries",
			Description: "List of entries.  Each entry can be a folder or a file.  If the entry is a folder, then it may recursively contain entries - if the recurse parameter was set to true",
			Type:        "object",
		},
	}
}

func (r *EditFile) Run(args []string) (any, error) {
	path := args[0]
	oldStr := args[1]
	newStr := args[2]
	fullpath, err := filepath.Abs(filepath.Join(r.ProjectRoot, path))
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fullpath)
	if err != nil {
		if os.IsNotExist(err) && oldStr == "" {
			return createNewFile(fullpath, newStr)
		}
		return "", err
	}

	oldContent := string(content)
	newContent := strings.Replace(oldContent, oldStr, newStr, -1)

	if oldContent == newContent && oldStr != "" {
		return "", fmt.Errorf("old_str not found in file")
	}

	err = os.WriteFile(fullpath, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}
