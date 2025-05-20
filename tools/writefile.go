package tools

import (
	"encoding/json"
	"fmt"
	"log"
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
			Required:    true,
		},
		{
			Name:        "encoding",
			Description: "This should be one of 'plain', 'base64' or 'json' to specify the encoding",
			Type:        "string",
			Required:    true,
		},
		{
			Name: "contents",
			Description: `
				The contents of file to be written to. 
				Depending on the encoding the following decoding is performed before contents are written to the file:

				"plain": 	File contents will be text and written verbatim.
				"base64":	File contents will be first base64 decoded and then written to.  Use this for saving binary content like images and media.
				"json":		JSON encoded string.   The JSON decoding of this string must result in a raw string.
			`,
			Type:     "string",
			Required: true,
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
	encoding := "json"
	if args["encoding"] != nil {
		encoding = args["encoding"].(string)
	}
	contents := args["contents"].(string)

	if encoding == "plain" {
		// do nothign
	} else if encoding == "base64" {
		// TODO
	} else {
		// json encoding
		var out string
		if err := json.Unmarshal([]byte(contents), &out); err != nil {
			log.Fatal("Cannot unmarshall json: ", err)
		} else {
			log.Println("Here.....")
			contents = out
		}
	}

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
