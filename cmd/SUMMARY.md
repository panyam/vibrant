## Go CLI Commands Summary (`./cmd`)

This directory contains the main application entry point and command-line interface (CLI) definitions for the `vibrant` tool, built using the `github.com/spf13/cobra` library.

### Purpose

*   Provides the user-facing commands to interact with the various functionalities of the `vibrant` application, including serving web interfaces and managing agent services.

### Key Components & Functionality

1.  **`main.go`**:
    *   The main entry point of the application.
    *   Simply calls `Execute()` from `root.go`.

2.  **`root.go`**:
    *   Defines the root `cobra.Command` (`rootCmd`) for the `vibrant` CLI.
    *   `Execute()`: This function is called by `main()` to run the CLI, parse arguments, and dispatch to the appropriate subcommand.
    *   `AddCommand(cmd *cobra.Command)`: A helper function to allow other files in the `cmd` package (like `canvas.go`, `agents.go`, `tools.go`) to register their commands with the `rootCmd`.

3.  **`canvas.go`**:
    *   Defines the `vibrant canvas` command.
    *   **Purpose**: Starts a web server to serve a frontend application (likely for UI, testing, and displaying Templar templates).
    *   Uses `github.com/panyam/templar/utils.BasicServer` to serve content.
    *   The server runs on port `7777` by default, configurable via the `-p` or `--port` flag.
    *   It uses `http.DefaultServeMux` implicitly, meaning other handlers can be registered globally to this mux before `BasicServer.Serve()` is called.
    *   **Current State**: This command is focused solely on serving the Templar frontend and does *not* include the agent WebSocket logic.

4.  **`agents.go`**:
    *   Defines the `vibrant agents` command group and its subcommand `vibrant agents serve`.
    *   **`vibrant agents serve` Purpose**: Starts a dedicated HTTP server for handling WebSocket connections and commands for the Chrome DevTools agent extension.
    *   Uses the `ServeMux` returned by `web.NewServeMux()` (from the `../web` package).
    *   This server runs on port `9999` by default, configurable via the `-p` or `--port` flag.
    *   The `web.NewServeMux()` handles its own routing for paths like `/agents/{connectionName}/subscribe` (WebSocket) and `/agents/{connectionName}/{COMMAND_TYPE}` (HTTP POST commands).

5.  **`tools.go`**:
    *   Defines the `vibrant tools` command group and its subcommands `json` and `run`.
    *   **Purpose**: Provides utilities for developers to interact with and test registered "tools" (likely programmatic functions or APIs that can be called, as defined in the `../tools` package).
    *   `vibrant tools` (no subcommand): Prints a list of available tools and their descriptions (calls `tools.PrintTools()`).
    *   `vibrant tools json`: Prints tool definitions in a JSON format, intended for consumption by an AI UX or other systems (calls `tools.ToolsJson()`).
    *   `vibrant tools run <TOOLNAME> [args...]`: Executes a specified tool with given arguments (calls `tools.RunTool(...)`). Input for tool parameters is expected via stdin after a prompt.

### Workflow Summary

*   The user runs `go run . <command> [subcommand] [flags]`.
*   `main.go` calls `root.Execute()`.
*   Cobra parses the command and flags, then executes the `Run` function of the matched command.
    *   `vibrant canvas`: Starts the Templar frontend server on port 7777.
    *   `vibrant agents serve`: Starts the WebSocket agent backend on port 9999.
    *   `vibrant tools ...`: Interacts with the tool management system.

### Project Structure & Dependencies

*   Commands are organized into separate files for clarity (e.g., `canvas.go`, `agents.go`).
*   Relies on the `../tools` package for tool definitions and execution logic.
*   Relies on the `../web` package for the agent WebSocket server's `ServeMux`.
*   Uses `github.com/panyam/templar/utils` for the `BasicServer` in the `canvas` command.
