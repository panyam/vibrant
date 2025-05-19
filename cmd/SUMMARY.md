## Go CLI Commands Summary (`./cmd`)

This directory contains the main application entry point and command-line interface (CLI) definitions for the `vibrant` tool, built using `github.com/spf13/cobra`.

### Purpose

*   Provides user-facing commands to interact with `vibrant`'s functionalities, including starting web servers and sending commands to connected browser clients via the agent server.

### Key Components & Functionality

1.  **`root.go`**:
    *   Defines the root `cobra.Command` (`rootCmd`) for `vibrant`.
    *   `Execute()`: Main entry point for the CLI.
    *   `AddCommand()`: Helper to register subcommands.
    *   **Persistent Flags**:
        *   `--client-id` (`-i`): Specifies the target client ID for commands. Defaults to the `VIBRANT_CLIENT_ID` environment variable if set. Stored in `rootCurrentClientId`.
        *   `--from-clipboard` (`-c`): A boolean flag (default `false`) indicating whether input for certain commands (like `client respond` or `client send`) should be read from the system clipboard instead of stdin. Stored in `rootFromClipboard`.
    *   `PersistentPreRunE`: Ensures `rootCurrentClientId` is correctly populated considering both the flag and the environment variable.

2.  **`main.go`**:
    *   Minimal entry point, calls `root.Execute()`.

3.  **`canvas.go`**:
    *   Defines `vibrant canvas`.
    *   Starts a web server (default port `7777`, configurable with `-p`) using `panyam/templar` for serving a frontend (likely HTML templates). This server is separate from the agent server.

4.  **`agents.go`**:
    *   Defines `vibrant agents serve`.
    *   Starts the dedicated HTTP/WebSocket agent server (default port `9999`, configurable with `-p`) using the `ServeMux` from the `../web` package. This server handles communication with the Chrome DevTools extension.

5.  **`client.go`**:
    *   Defines the `vibrant client` command group for sending instructions to a connected Chrome DevTools extension instance via the agent server.
    *   **`sendEvalScript(scriptToEvaluate string, waitForResult bool)` function**:
        *   The core function for sending commands. It uses the `rootCurrentClientId` (from root persistent flag/env var).
        *   Makes an HTTP POST request to `http://localhost:9999/agents/{clientId}/eval`.
        *   The **body of this POST request is the raw JavaScript string** (`scriptToEvaluate`).
        *   It can optionally wait for the script's result from the server if `waitForResult` is true (by appending `?wait=true` to the URL).
    *   **Subcommands**:
        *   `scrolltop`, `scrollbottom`, `scrolldelta [deltaY]`: Send predefined JavaScript snippets for page scrolling actions (fire-and-forget).
        *   `gettitle`: Sends a script to get `document.title` and waits for/prints the result.
        *   `respond`, `send`: These commands are more application-specific.
            *   They read input (from stdin or clipboard, using `rootFromClipboard`).
            *   Use `buildSetInputValueScriptUsingTemplate` to construct a JavaScript string. This script sets the value of a specific textarea (e.g., `ms-function-call-chunk textarea` or `ms-prompt-input-wrapper textarea`) and, if the local `--submit` flag is true (defaults to true), also clicks a corresponding submit button.
            *   These commands use `sendEvalScript` with `waitForResult = true` to get a report back from the script execution.
    *   **`buildSetInputValueScriptUsingTemplate`**: Uses Go's `text/template` package to generate the JavaScript for setting input values and optionally clicking submit buttons. It JSON-marshals string inputs (selector, value) before injecting them into the template to ensure they are valid JS string literals.

6.  **`tools.go`**:
    *   Defines `vibrant tools` and subcommands (`json`, `run`) for interacting with a system of developer tools defined in the `../tools` package.

### Workflow Summary

*   User runs `go run . [global flags] <command> [subcommand] [local flags]`.
*   `root.go` parses global persistent flags (`--client-id`, `--from-clipboard`).
*   The appropriate command's `Run` function is executed.
    *   `client` subcommands construct JavaScript strings (some using Go templates) and use `sendEvalScript` to POST them to the `/agents/{clientId}/eval` endpoint of the agent server.
    *   The agent server (`web/server.go`) then relays these scripts to the Chrome extension for execution.
