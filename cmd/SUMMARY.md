## Go CLI Commands Summary (`./cmd`)

This directory contains the main application entry point and command-line interface (CLI) definitions for the `vibrant` tool, built using `github.com/spf13/cobra`.

### Purpose

*   Provides user-facing commands to interact with `vibrant`'s functionalities, including starting web servers, sending commands to connected browser clients via the agent server, and managing local tools.

### Key Components & Functionality

1.  **`root.go`**:
    *   Defines the root `cobra.Command` (`rootCmd`) for `vibrant`.
    *   `Execute()`: Main entry point for the CLI.
    *   `AddCommand()`: Helper to register subcommands.
    *   **Persistent Flags**:
        *   `--client-id` (`-i`): Specifies the target client ID for commands. Defaults to the `VIBRANT_CLIENT_ID` environment variable if set. Stored in `rootCurrentClientId`.
        *   `--from-clipboard` (`-c`): A boolean flag (default `false`) indicating whether input for certain commands should be read from the system clipboard. Stored in `rootFromClipboard`.
    *   `PersistentPreRunE`: Ensures `rootCurrentClientId` is correctly populated.

2.  **`main.go`**:
    *   Minimal entry point, calls `root.Execute()`.

3.  **`js.go`**:
    *   **`sendEvalScript(scriptToEvaluate string, waitForResult bool)` function**: The core function for sending JavaScript evaluation commands. It uses `rootCurrentClientId`. Makes an HTTP POST request to `http://localhost:9999/agents/{clientId}/eval` with the raw JavaScript string as the body. Can optionally wait for the script's result.
    *   **`buildSetInputValueScriptUsingTemplate`**: Helper to generate JavaScript for setting input values using a Go template.

4.  **`client.go`**:
    *   Defines the `vibrant client` command group for sending instructions to a connected Chrome DevTools extension.
    *   **Subcommands**:
        *   `scrolltop`, `scrollbottom`, `scrolldelta [deltaY]`: Send predefined JavaScript snippets for page scrolling.
        *   `gettitle`: Sends a script to get `document.title` and waits for/prints the result.
    *   The `init()` function in this file also registers commands defined in other `client_*.go` files (like `screenshot.go`).

5.  **`screenshot.go` (New)**:
    *   Defines the `vibrant screenshot` command.
    *   **Purpose**: Captures screenshots of specified DOM elements on the inspected page.
    *   **Flags**:
        *   `--selector` (`-s`): (Repeatable) CSS selector of an element to screenshot.
        *   `--output-dir` (`-o`): Directory to save the screenshots (defaults to `./screenshots`).
    *   **Workflow**:
        1.  Collects selectors and validates inputs.
        2.  Makes an HTTP POST request to `http://localhost:9999/agents/{clientId}/screenshots?wait=true`. The request body is a JSON `{"selectors": ["selector1", ...]}`.
        3.  Receives a JSON response containing a map where keys are selectors and values are base64 encoded PNG data URLs (or null for errors).
        4.  Decodes each data URL and saves it as a `.png` file in the output directory.

6.  **`calls.go`**:
    *   Defines `vibrant calls` and subcommands (`list`, `respond`).
    *   `list`: Lists pending tool calls from an AI interface by evaluating JavaScript to find them on the page.
    *   `respond`: Responds to a specific tool call by running a local tool (from the `../tools` package) with parameters from the call, then injects the result back into the AI interface page via JavaScript.
    *   Uses `sendEvalScript` with `waitForResult = true` and custom Go templates for script generation.

7.  **`send.go`**:
    *   Defines `vibrant send` (previously `calls sendprompt`).
    *   Sets a value in a specific textarea (e.g., a prompt input field on an AI page) and optionally submits it, using `buildSetInputValueScriptUsingTemplate` and `sendEvalScript`.

8.  **`canvas.go`**:
    *   Defines `vibrant canvas`.
    *   Starts a web server (default port `7777`) using `panyam/templar` for serving a frontend (likely HTML templates from `./templates` and JS from `./components`). This server is separate from the agent server.

9.  **`agents.go`**:
    *   Defines `vibrant agents serve`.
    *   Starts the dedicated HTTP/WebSocket agent server (default port `9999`) using the `ServeMux` from the `../web` package. This server handles communication with the Chrome DevTools extension.

10. **`tools.go`**:
    *   Defines `vibrant tools` and subcommands (`json`, `run`) for interacting with a system of developer tools defined in the `../tools` package.

### Workflow Summary for Core Operations

*   User runs `go run . [global flags] <command> [subcommand] [local flags]`.
*   `root.go` parses global persistent flags.
*   The appropriate command's `Run` function is executed.
    *   `client` subcommands typically use `sendEvalScript` (for JS evaluation) or make specific HTTP requests (like `screenshot`) to the agent server.
    *   The agent server (`web/server.go`) then relays instructions (as JavaScript or specialized commands like screenshot requests) to the Chrome extension for execution.
