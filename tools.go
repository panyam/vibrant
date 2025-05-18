package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	Run(args map[string]any) (any, error)
}

type BaseFileTool struct {
	ProjectRoot string
}

func (b *BaseFileTool) ResolvePath(path string) (fullpath string, err error) {
	fullpath, err = filepath.Abs(filepath.Join(b.ProjectRoot, path))
	return
}

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

type ApplyFileDiff struct {
	BaseFileTool
}

func (r *ApplyFileDiff) Name() string {
	return "apply_file_diff"
}

func (r *ApplyFileDiff) Description() string {
	return `Applies a file diff onto an existing file and returns a success or a failure along with the contents of the file being updated incase the diff being applied is based on an older version of the file`
}

func (r *ApplyFileDiff) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "path",
			Description: "Path of the file to create or overwrite.",
			Type:        "string",
		},
		{
			Name:        "diff",
			Description: "Unix still diff/patch to apply to a file.  If the diff is invalid (for example it is based on an older version of the file) then an error will be thrown",
			Type:        "string",
		},
	}
}

func (r *ApplyFileDiff) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "result",
			Description: "Error or Number of bytes written",
			Type:        "object",
		},
	}
}

func (r *ApplyFileDiff) Run(args map[string]any) (any, error) {
	path := args["path"].(string)
	diff := args["diff"].(string)
	fullpath, err := r.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	// TODO - replace with stat
	currcontents, err := os.ReadFile(fullpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return "", err
	}

	tempinfile, err := os.CreateTemp("", "inputfile")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tempinfile.Name())
	temppatchfile, err := os.CreateTemp("", "patchfile")
	if err != nil {
		panic(err)
	}
	defer os.Remove(temppatchfile.Name())

	if _, err = tempinfile.Write([]byte(currcontents)); err != nil {
		panic(err)
	}

	if _, err = temppatchfile.Write([]byte(diff)); err != nil {
		panic(err)
	}

	cmd := exec.Command("patch", "-u", tempinfile.Name(), "-i", temppatchfile.Name())
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	log.Println("Patch result: ", output)
	return "SUCCESS", nil
}
