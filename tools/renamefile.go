package tools

import (
	"fmt"
	"os"
)

type RenameFile struct {
	BaseFileTool
}

func (r *RenameFile) Name() string {
	return "rename_file"
}

func (r *RenameFile) Description() string {
	return `Renames/Moves a source file to its new destination`
}

func (r *RenameFile) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "src",
			Description: "Source path of the file to be renamed/moved",
			Type:        "string",
			Required:    true,
		},
		{
			Name:        "dest",
			Description: "Destination path to move/rename the file to",
			Type:        "string",
			Required:    true,
		},
	}
}

func (r *RenameFile) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "result",
			Description: "Error or Success",
			Type:        "string",
		},
	}
}

func (r *RenameFile) Run(args map[string]any) (any, error) {
	srcpath := args["src"].(string)
	fullsrcpath, err := r.ResolvePath(srcpath)
	if err != nil {
		return nil, err
	}
	destpath := args["dest"].(string)
	fulldestpath, err := r.ResolvePath(destpath)
	if err != nil {
		return nil, err
	}
	err = os.Rename(fullsrcpath, fulldestpath)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully renamed to %s to %s", srcpath, destpath), nil
}
