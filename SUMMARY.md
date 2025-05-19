# Project `vibrant` - Overall Summary

## Overview

`vibrant` is a Go-based toolchain featuring a CLI and a Chrome DevTools extension. It enables a backend Go "agent" to remotely execute JavaScript on an inspected webpage, facilitating advanced development, debugging, and potentially AI-driven web interactions.

## Current State & Key Features

### 1. Go Application (Backend & CLI)

*   **Core Architecture**: The system revolves around a "remote evaluation" model. The Go backend sends JavaScript snippets to the Chrome extension, which executes them in the context of the inspected page and returns results.
*   **CLI (`./cmd`)**: Built with Cobra.
    *   **Persistent Flags (`root.go`)**:
        *   `--client-id` (`-i`): Specifies the target browser client instance. Defaults to `VIBRANT_CLIENT_ID` environment variable.
        *   `--from-clipboard` (`-c`): Allows certain commands to take input from the clipboard.
    *   **`vibrant canvas` (`canvas.go`)**: Starts a frontend web server (default port `7777`) using `panyam/templar` for UI and template rendering. This is separate from the agent server.
    *   **`vibrant agents serve` (`agents.go`)**: Starts the dedicated agent communication server (default port `9999`) using components from the `../web` package.
    *   **`vibrant client` (`client.go`)**: A command group to send instructions (as JavaScript to be evaluated) to a specific Chrome extension client via the agent server's `/agents/{clientId}/eval` HTTP POST endpoint.
        *   Subcommands like `scrolltop`, `scrollbottom`, `scrolldelta`, `gettitle` send predefined or dynamically generated JavaScript.
        *   Commands like `respond` and `send` construct more complex JavaScript (using Go templates) to set input values on specific page elements and optionally click submit buttons. These can take input from stdin or the clipboard.
        *   Supports both fire-and-forget script evaluations and waiting for results (`?wait=true` on the `/eval` endpoint).
    *   **`vibrant tools` (`tools.go`)**: Utilities to list, describe (JSON), and run developer tools defined in the `../tools` package.
*   **Agent Web Service (`./web/server.go`)**:
    *   Uses `panyam/goutils/http` for WebSocket connection management and `panyam/goutils/conc` for message fanout.
    *   **WebSocket Endpoint (`GET /agents/{clientId}/subscribe`)**: Where the Chrome extension connects. Upon connection, it sends an initial "welcome" `EVALUATE_SCRIPT` message to the client.
    *   **Script Evaluation Endpoint (`POST /agents/{clientId}/eval`)**:
        *   Accepts a **raw JavaScript string** in the POST body.
        *   Internally generates a `requestId`.
        *   Wraps the script and `requestId` into an `EVALUATE_SCRIPT` JSON message and broadcasts it to the specified `clientId` via WebSocket.
        *   If `?wait=true` is in the URL, it blocks and waits for an `EVALUATION_RESULT` from the client (with a timeout), then includes this result in its HTTP response.
    *   **Result Handling**: The WebSocket handler (`Conn.HandleMessage`) listens for `EVALUATION_RESULT` messages from the extension, correlates them using `requestId`, and forwards results to any waiting HTTP `/eval` handler via a channel.
*   **Developer Tools (`./tools`)**:
    *   A framework for tools like `read_file`, `list_files`, `create_file`, `apply_file_diff`.
    *   `tools.GetInputFromUserOrClipboard` helper used by `cmd/client.go`.

### 2. Chrome DevTools Extension (`./plugins/chrome`)

*   **Thin Client for Remote Evaluation**: The panel (`panel.js`) is now primarily a conduit for executing JavaScript sent by the Go backend.
*   **DevTools Panel ("Agent Logger")**:
    *   Provides UI for connecting to an agent server by name.
    *   Styled to respect Chrome DevTools light/dark themes.
    *   Includes auto-reconnection logic for the WebSocket connection with exponential backoff.
*   **Message Flow**:
    1.  Go agent server sends `{"type": "EVALUATE_SCRIPT", "requestId": "...", "scriptToEvaluate": "..."}` via WebSocket.
    2.  Extension's `background.js` relays this to `panel.js`.
    3.  `panel.js` uses `chrome.devtools.inspectedWindow.eval(scriptToEvaluate, callback)`.
    4.  The callback in `panel.js` constructs `{"type": "EVALUATION_RESULT", "requestId": "...", "result": ..., "isException": ..., "exceptionInfo": ...}`.
    5.  This result is sent via `background.js` back to the Go agent server over WebSocket.

## Project Structure

*   **`./cmd`**: Go CLI (root, canvas, agents, client, tools).
*   **`./web`**: Go backend for agent WebSocket/HTTP server.
*   **`./tools`**: Go framework and implementations for developer tools.
*   **`./plugins/chrome`**: Chrome DevTools extension.
*   Other files: `go.mod`, `Makefile`, frontend build configs.

## How to Run

1.  **Agent Server**: `go run . agents serve` (on `localhost:9999`).
2.  **Canvas/UI Server** (if needed): `go run . canvas` (on `localhost:7777`).
3.  **Chrome Extension**: Load unpacked from `./plugins/chrome`.
4.  **Interact**:
    *   Open DevTools, connect the "Agent Logger" panel (e.g., to client "testClient").
    *   Use `vibrant client -i testClient <subcommand>` to send commands.
    *   Use `POST` requests to `http://localhost:9999/agents/testClient/eval` (body is raw JS) or `GET /test_eval?agent=testClient&script=...`.
