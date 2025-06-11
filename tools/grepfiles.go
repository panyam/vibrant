package tools

import (
	"os"
	"path/filepath"
)

type GrepFiles struct {
	BaseFileTool
}

func (r *GrepFiles) Name() string {
	return "grep_files"
}

func (r *GrepFiles) Description() string {
	return `
	A direct tool wrapper over unix Grep utility.   Use this tool to search for content in files with several options that grep provides.
	`
}

func (r *GrepFiles) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "commandline",
			Description: "Full grep command line.",
			Type:        "string",
			Required:    true,
		},
	}
}

func (r *GrepFiles) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "result",
			Description: "The result of the grep listing",
			Type:        "object",
		},
	}
}

func (r *GrepFiles) Run(args map[string]any) (any, error) {
	path := "."
	if args != nil && args["path"] != nil {
		path = args["path"].(string)
	}
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
