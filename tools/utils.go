package tools

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

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

func RunCommand(fullCommand, workingDir string) (stdout io.Reader, stderr io.Reader, stdall io.Reader) {
	// Parse the command - split on spaces (basic parsing)
	parts := strings.Fields(fullCommand)
	if len(parts) == 0 {
		// Return empty readers for invalid command
		return strings.NewReader(""), strings.NewReader(""), strings.NewReader("")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = workingDir

	// Buffers to capture output
	var stdoutBuf, stderrBuf, combinedBuf bytes.Buffer

	// Create writers that write to both individual buffers and combined buffer
	stdoutWriter := io.MultiWriter(&stdoutBuf, &combinedBuf)
	stderrWriter := io.MultiWriter(&stderrBuf, &combinedBuf)

	// Use a mutex to ensure combined output is written in correct order
	var mu sync.Mutex

	// Wrap the writers with mutex protection for the combined buffer
	cmd.Stdout = &synchronizedWriter{writer: stdoutWriter, mu: &mu}
	cmd.Stderr = &synchronizedWriter{writer: stderrWriter, mu: &mu}

	// Run the command
	cmd.Run() // Ignoring error since we're capturing stderr anyway

	// Return readers for the captured output
	return strings.NewReader(stdoutBuf.String()),
		strings.NewReader(stderrBuf.String()),
		strings.NewReader(combinedBuf.String())
}

// synchronizedWriter wraps an io.Writer with mutex protection
type synchronizedWriter struct {
	writer io.Writer
	mu     *sync.Mutex
}

func (sw *synchronizedWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.writer.Write(p)
}
