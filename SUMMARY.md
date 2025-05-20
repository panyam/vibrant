# Project `vibrant` - Overall Summary

## Overview

`vibrant` is a Go-based toolchain designed to bridge a command-line interface (CLI) with a Chrome DevTools extension. Its primary purpose is to allow a Go backend (the "agent server") to remotely execute JavaScript, perform actions like element-specific screenshots, and simulate user interactions like pasting content on a webpage being inspected by Chrome DevTools. This enables advanced automation, debugging, and programmatic interaction with web pages from the Go environment.

## Current State & Key Features

### 1. Go Application (Backend & CLI)

*   **Core Architecture**: The system revolves around a remote command execution model. The Go backend sends instructions (JavaScript snippets or specialized commands) to the Chrome extension, which executes them in the context of the inspected page and returns results or data.
*   **CLI (`./cmd`)**: Built with Cobra.
    *   **Persistent Flags (`root.go`)**: `--client-id` (`-i`) for target browser client, `--from-clipboard` (`-c`) for clipboard input.
    *   **`vibrant agents serve` (`agents.go`)**: Starts the dedicated agent communication server (default port `9999`) using components from the `../web` package.
    *   **`vibrant client` (`client.go`, etc.)**: A command group to send instructions to a Chrome extension client via the agent server.
        *   `eval` related commands (`gettitle`, `scrolltop`, etc.): Send JavaScript to be evaluated via the agent server's `/agents/{clientId}/eval` HTTP POST endpoint.
        *   Supports both fire-and-forget operations and waiting for results (`?wait=true` on relevant endpoints).
    *   `vibrant screenshot` command: Captures screenshots of specified DOM elements. Communicates with `/agents/{clientId}/screenshots`.
    *   `vibrant paste` command (New): Simulates pasting content (e.g., an image from a file or data URL) into a specified DOM element. Communicates with `/agents/{clientId}/paste`.
    *   **`vibrant calls` (`calls.go`)**: Commands to list and respond to tool calls, often from an AI interface, by running local tools (`../tools`) and injecting results back into the page.
    *   **`vibrant tools` (`tools.go`)**: Utilities to list, describe (JSON), and run local developer tools defined in the `../tools` package.
    *   **`vibrant canvas` (`canvas.go`)**: Starts a separate frontend web server.

*   **Agent Web Service (`./web`)**: Implemented in `web/server.go`.
    *   **WebSocket Endpoint (`GET /agents/{clientId}/subscribe`)**: Chrome extension connects here.
    *   **Script Evaluation Endpoint (`POST /agents/{clientId}/eval`)**: Accepts raw JavaScript, sends `EVALUATE_SCRIPT` via WebSocket.
    *   **Element Screenshot Endpoint (`POST /agents/{clientId}/screenshots`)**: Accepts JSON `{"selectors": [...]}`. Sends `CAPTURE_ELEMENTS_SCREENSHOT` via WebSocket.
    *   **Paste Data Endpoint (`POST /agents/{clientId}/paste`) (New)**: Accepts JSON `{"selector": "...", "dataUrl": "..."}`. Sends `PASTE_DATA` via WebSocket.
    *   **Result Handling**: The WebSocket handler (`Conn.HandleMessage`) listens for `EVALUATION_RESULT`, `ELEMENTS_SCREENSHOT_RESULT`, or `PASTE_RESULT` messages from the extension, correlates them using a `requestId`, and forwards results to waiting HTTP handlers.

*   **Developer Tools (`./tools`)**: A framework for local file system tools like `read_file`, `list_files`, `create_file`, `apply_file_diff`.

### 2. Chrome DevTools Extension (`./plugins/chrome`)

*   **Core Function**: Acts as the execution bridge on the client-side.
*   **`manifest.json`**: Defines permissions including `storage`, `tabs`, `activeTab`, `clipboardRead`, `clipboardWrite`, and host permissions for `ws://localhost:9999/*` and `<all_urls>`.
*   **`background.js` (Service Worker)**: Manages WebSocket connections and relays messages. Handles `REQUEST_TAB_CAPTURE` for screenshots.
*   **`panel.js` (DevTools Panel "Agent Logger")**: 
    *   UI for connection management.
    *   Receives `EVALUATE_SCRIPT` messages: executes JS using `chrome.devtools.inspectedWindow.eval()` and sends `EVALUATION_RESULT` back.
    *   Receives `CAPTURE_ELEMENTS_SCREENSHOT` messages: gets bounding boxes, requests full tab capture, crops images, and sends `ELEMENTS_SCREENSHOT_RESULT` back.
    *   Receives `PASTE_DATA` messages (New): executes a script in the inspected page to find the element, fetch the data URL to a Blob, create a `DataTransfer` object, and dispatch a synthetic `paste` event on the element. Sends `PASTE_RESULT` back.

## Project Structure

*   **`./cmd`**: Go CLI.
*   **`./web`**: Go backend for the agent WebSocket/HTTP server.
*   **`./tools`**: Go framework for local developer tools.
*   **`./plugins/chrome`**: Chrome DevTools extension.
*   **`./templates`**: Go HTML templates.
*   Other files: `go.mod`, `Makefile`, etc.

## How to Run (Core Agent Functionality)

1.  **Agent Server**: `go run ./cmd agents serve`.
2.  **Chrome Extension**: Load unpacked from `./plugins/chrome`. Reload and **close/reopen DevTools** after changes.
3.  **Interact**: 
    *   Open DevTools on a target page, open "Agent Logger" panel, connect with a client ID.
    *   Use `vibrant client -i <clientID> <subcommand>` (e.g., `gettitle`, `screenshot -s "h1"`, `paste -s "#myInput" -f "image.png"`).
