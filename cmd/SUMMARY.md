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
    *   The `init()` function in this file also registers commands defined in other `client_*.go` files (like `screenshot.go` and `client_paste.go`).

5.  **`screenshot.go`**:
    *   Defines the `vibrant screenshot` command.
    *   **Purpose**: Captures screenshots of specified DOM elements on the inspected page.
    *   **Flags**:
        *   `--selector` (`-s`): (Repeatable) CSS selector of an element to screenshot.
        *   `--output-dir` (`-o`): Directory to save the screenshots (defaults to `./screenshots`).
    *   **Workflow**:
        1.  Collects selectors and validates inputs.
        2.  Makes an HTTP POST request to `http://localhost:9999/agents/{clientId}/screenshots?wait=true`. The request body is a JSON `{"selectors": ["selector1", ...]}`.
        3.  Receives a JSON response where the `response` field contains a map of selectors to base64 encoded PNG data URLs (or null/error for issues).
        4.  Decodes each data URL and saves it as a `.png` file in the output directory.

6.  **`client_paste.go` (New)**:
    *   Defines the `vibrant client paste` command.
    *   **Purpose**: Simulates a paste event on a target DOM element, allowing content (especially images from files or data URLs) to be "pasted".
    *   **Flags**:
        *   `--selector` (`-s`): (Required) CSS selector of the target element.
        *   `--data` (string): Base64 data URL of the content to paste.
        *   `--file` (`-f`): Path to an image file to paste (CLI converts to data URL).
        *   One of `--data` or `--file` must be provided.
    *   **Workflow**:
        1.  Validates inputs. If `--file` is used, reads the file and constructs a data URL.
        2.  Makes an HTTP POST request to `http://localhost:9999/agents/{clientId}/paste?wait=true`.
        3.  Request body is JSON: `{"selector": "css-selector", "dataUrl": "data:..."}`.
        4.  Receives a JSON response indicating success or failure of dispatching the paste event.

7.  **`calls.go`**:
    *   Defines `vibrant calls` and subcommands (`list`, `respond`).
    *   `list`: Lists pending tool calls from an AI interface by evaluating JavaScript to find them on the page.
    *   `respond`: Responds to a specific tool call by running a local tool (from the `../tools` package) with parameters from the call, then injects the result back into the AI interface page via JavaScript.
    *   Uses `sendEvalScript` with `waitForResult = true` and custom Go templates for script generation.

8.  **`send.go`**:
    *   Defines `vibrant send`.
    *   Sets a value in a specific textarea and optionally submits it, using `buildSetInputValueScriptUsingTemplate` and `sendEvalScript`.

9.  **`canvas.go`**:
    *   Defines `vibrant canvas`.
    *   Starts a web server (default port `7777`) for a frontend UI.

10. **`agents.go`**:
    *   Defines `vibrant agents serve`.
    *   Starts the dedicated HTTP/WebSocket agent server (default port `9999`).

11. **`tools.go`**:
    *   Defines `vibrant tools` for interacting with local developer tools.

### Workflow Summary for Core Operations

*   User runs `go run . [global flags] <command> [subcommand] [local flags]`.
*   `root.go` parses global persistent flags.
*   The appropriate command's `Run` function is executed.
    *   `client` subcommands typically make specific HTTP requests (like `screenshot`, `paste`) or use `sendEvalScript` (for general JS evaluation) to the agent server.
    *   The agent server (`web/server.go`) then relays instructions (as JavaScript or specialized commands like `PASTE_DATA` or screenshot requests) to the Chrome extension for execution.
