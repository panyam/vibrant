package tools

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"golang.design/x/clipboard"
)

func createNewFile(filePath, content string) (string, error) {
	dir := path.Dir(filePath)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file %s", filePath), nil
}

func getUserMessageTillEOF() (string, error) {
	// Create a new reader from stdin
	reader := bufio.NewReader(os.Stdin)

	// Read all input until EOF
	var input strings.Builder
	var buffer []byte = make([]byte, 8192, 8192)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			input.Write(buffer[:n])
		}
		if err != nil {
			if err == io.EOF {
				// reached the end
				err = nil
			}
			return input.String(), err
		}
	}
}

func GetInputFromUserOrClipboard(fromClipboard bool, prompt string) (input string, err error) {
	if fromClipboard {
		input = string(clipboard.Read(clipboard.FmtText))
	} else {
		if prompt == "" {
			prompt = "Enter Prompt"
		}
		fmt.Printf("\u001b[94m%s\u001b[0m: ", prompt)
		input, err = getUserMessageTillEOF()
	}
	return
}
