package tools

import (
	"io"
	"path/filepath"
)

type RunShellCommand struct {
	BaseFileTool
}

func (r *RunShellCommand) Name() string {
	return "run_shell_command"
}

func (r *RunShellCommand) Description() string {
	return `
	Lists the files in a given folder provided by the 'path' parameter.   The 'recurse' parameter enables recursive file listing.

	Returns an error or a list of entries.  Each entry is a file or a folder.   Folder entries could recursively contain other entries.
	`
}

func (r *RunShellCommand) Parameters() []*Parameter {
	return []*Parameter{
		{
			Name:        "command",
			Description: "The command along with all the arguments to run.  Any files should be relative to the working_dir parameter",
			Type:        "string",
			Required:    true,
		},
		{
			Name:        "working_dir",
			Description: "Directory from which to run the command.  If this is not specified then '.' is assumed and the command will be run from the project's root directory",
			Type:        "boolean",
		},
	}
}

func (r *RunShellCommand) Returns() []*Parameter {
	return []*Parameter{
		{
			Name:        "output",
			Description: "Returns all output - including standard output and standard error",
			Type:        "object",
		},
	}
}

func (r *RunShellCommand) Run(args map[string]any) (any, error) {
	working_dir := "."
	if args != nil && args["working_dir"] != nil {
		working_dir = args["working_dir"].(string)
	}
	dir, err := filepath.Abs(filepath.Join(r.ProjectRoot, working_dir))
	if err != nil {
		return nil, err
	}
	if args["command"] == "" || args["command"] == nil {
		return "Please provide a command to execute", nil
	}
	cmd := args["command"].(string)

	_, _, combined := RunCommand(cmd, dir)
	output, err := io.ReadAll(combined)
	return output, err
}
