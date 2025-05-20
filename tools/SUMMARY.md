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
    *   **`RunTool(fromClipboard bool, name string, params map[string]any)`**: 
        *   If `params` is nil, reads JSON input from the user or clipboard.
        *   Retrieves the tool by `name` from the registry.
        *   Executes the tool's `Run` method with the parsed parameters.
        *   Prints the result or error and copies string results to the clipboard.
    *   **`ToolsJson()`**: Serializes the definitions (name, description, parameters) of all registered tools into a JSON array and prints it. This is intended for consumption by external systems (e.g., an AI UX).
    *   **`PrintTools()`**: Prints a formatted list of all tools, their descriptions, parameters, and return types to the console.
    *   **`GetInputFromUserOrClipboard(fromClipboard bool, prompt string)`**: Helper function (from `utils.go`) to read input either from stdin or the system clipboard.

3.  **Concrete Tool Implementations** (each in its own file):
    *   **`readfile.go` (`ReadFile` tool)**: Reads file content.
    *   **`listfiles.go` (`ListFiles` tool)**: Lists files/directories.
    *   **`writefile.go` (`WriteFile` tool)**: Creates/overwrites a file.
    *   **`applydiff.go` (`ApplyFileDiff` tool)**: Applies a diff using the system `patch` command.

4.  **`utils.go`**:
    *   Contains helper functions like `getUserMessageTillEOF` and `createNewFile`, and `GetInputFromUserOrClipboard`.

### Current Tool Implementations

*   `read_file`
*   `list_files`
*   `write_file`

### Workflow Summary

*   Tools implement the `Tool` interface and are registered in `runner.go`.
*   `vibrant tools` CLI commands (defined in `../cmd/tools.go`) use functions from `runner.go` to manage and execute these tools.
*   The `vibrant calls respond` command also uses `RunTool` to fulfill tool requests originating from an AI interface.
