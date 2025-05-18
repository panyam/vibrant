## Go Tools Package Summary (`./tools`)

This directory contains the Go package responsible for defining, managing, and executing a set of developer tools. These tools are designed to interact with the file system and potentially other project aspects, and can be invoked via the `vibrant tools` CLI commands.

### Purpose

*   Provide a framework for defining tools with names, descriptions, parameters, and return types.
*   Implement concrete tools for common development tasks (file reading, listing, creation, diff application).
*   Offer utilities to run these tools and present their definitions.

### Key Components & Functionality

1.  **`core.go`**:
    *   **`Parameter` struct**: Defines the structure for tool parameters and return values (Name, Description, Type).
    *   **`Tool` interface**: The central interface that all tools must implement.
        *   `Name() string`: Returns the tool's unique name.
        *   `Description() string`: Provides a human-readable description.
        *   `Parameters() []*Parameter`: Lists the input parameters the tool expects.
        *   `Returns() []*Parameter`: Describes the output the tool produces.
        *   `Run(args map[string]any) (any, error)`: Executes the tool's logic with provided arguments.
    *   **`BaseFileTool` struct**: A utility struct that can be embedded in file-related tools.
        *   `ProjectRoot string`: Specifies the root directory against which relative paths are resolved.
        *   `ResolvePath(path string) (string, error)`: Converts a relative path to an absolute path based on `ProjectRoot`.

2.  **`runner.go`**:
    *   **Tool Registry (`tools map[string]Tool`)**: A global map initialized in `init()` to register all available tool implementations.
    *   **`RunTool(name string, args []string)`**: 
        *   (Currently expects tool parameters to be provided via stdin as a JSON map after a prompt).
        *   Retrieves the tool by `name` from the registry.
        *   Executes the tool's `Run` method with the parsed parameters.
        *   Prints the result or error.
    *   **`ToolsJson()`**: Serializes the definitions (name, description, parameters) of all registered tools into a JSON array and prints it. This is intended for consumption by external systems (e.g., an AI UX).
    *   **`PrintTools()`**: Prints a formatted list of all tools, their descriptions, parameters, and return types to the console.

3.  **Concrete Tool Implementations** (each in its own file):
    *   **`readfile.go` (`ReadFile` tool)**:
        *   Reads the content of a specified file.
        *   Path is relative to `ProjectRoot`.
    *   **`listfiles.go` (`ListFiles` tool)**:
        *   Lists files and directories within a given path (relative to `ProjectRoot`).
        *   Currently implements a non-recursive listing (the `recurse` parameter defined in its `Parameters()` method is not fully utilized in the `Run` logic as it uses `filepath.Walk` which is inherently recursive if not stopped).
    *   **`createfile.go` (`CreateFile` tool)**:
        *   Creates or overwrites a file with specified content.
        *   Path is relative to `ProjectRoot`.
    *   **`applydiff.go` (`ApplyFileDiff` tool)**:
        *   Applies a Unix-style diff/patch to an existing file.
        *   Uses the system `patch` command via `os/exec`.
        *   Writes the current file content and the diff to temporary files before applying.

4.  **`utils.go`**:
    *   **`createNewFile(filePath, content string)`**: Helper to create directories if needed and write file content.
    *   **`getUserMessageTillEOF()`**: Reads multi-line input from stdin until EOF, used by `RunTool` to get JSON parameters.
    *   **`getUserMessage()`**: (Not currently used by `RunTool`) Reads a single line from stdin.

### Current Tool Implementations

*   `read_file`: Reads a file.
*   `list_files`: Lists files in a directory (currently walks all subdirectories).
*   `create_file`: Creates or overwrites a file.
*   `apply_file_diff`: Applies a diff to a file using the `patch` command.

### Workflow Summary

*   Tools are defined by implementing the `Tool` interface.
*   They are registered in the `tools` map in `runner.go`.
*   The `vibrant tools` CLI commands (defined in `../cmd/tools.go`) use functions from `runner.go` to list, describe, or execute these tools.
*   For tool execution, parameters are currently passed as a JSON string via stdin.
