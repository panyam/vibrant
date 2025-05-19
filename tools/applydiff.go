package tools

import (
	"io"
	"log"
	"os"
	"os/exec"
)

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
	log.Println("Creating Temporary Input File: ", tempinfile.Name())
	temppatchfile, err := os.CreateTemp("", "patchfile")
	if err != nil {
		panic(err)
	}
	log.Println("Creating Temporary Patch File: ", temppatchfile.Name())

	if _, err = tempinfile.Write([]byte(currcontents)); err != nil {
		panic(err)
	}

	if _, err = temppatchfile.Write([]byte(diff)); err != nil {
		panic(err)
	}

	log.Printf("Running patch -u %s -i %s", tempinfile.Name(), temppatchfile.Name())
	cmd := exec.Command("patch", "-u", tempinfile.Name(), "-i", temppatchfile.Name())
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Println("Cannot get stderr: ", err)
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Cannot get stdout: ", err)
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		log.Fatal("Could not sart command: ", err)
	}

	output, err := io.ReadAll(stdout)
	errout, err := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		log.Println("Error patching: ", err.Error(), errout)
		log.Println("Error Output: ", string(errout))
		log.Println("Std Output: ", string(output))
		log.Println("Could not wait: ", err)
		if string(errout) != "" {
			return string(errout), nil
		}
		return string(output), nil
	}

	log.Println("Patch result: ", output)
	return "SUCCESS", nil
}
