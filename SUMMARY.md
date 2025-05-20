# Project `vibrant` - Overall Summary

## Overview

`vibrant` is a Go-based toolchain designed to bridge a command-line interface (CLI) with a Chrome DevTools extension. Its primary purpose is to allow a Go backend (the "agent server") to remotely execute JavaScript and perform other actions (like taking element-specific screenshots) on a webpage being inspected by Chrome DevTools. This enables advanced automation, debugging, and programmatic interaction with web pages from the Go environment.

## Current State & Key Features

### 1. Go Application (Backend & CLI)

*   **Core Architecture**: The system revolves around a remote command execution model. The Go backend sends instructions (JavaScript snippets or specialized commands) to the Chrome extension, which executes them in the context of the inspected page and returns results or data.
*   **CLI (`./cmd`)**: Built with Cobra.
    *   **Persistent Flags (`root.go`)**: `--client-id` (`-i`) for target browser client, `--from-clipboard` (`-c`) for clipboard input.
    *   **`vibrant agents serve` (`agents.go`)**: Starts the dedicated agent communication server (default port `9999`) using components from the `../web` package.
    *   **`vibrant client` (`client.go`, `client_screenshot.go`, etc.)**: A command group to send instructions to a Chrome extension client via the agent server.
        *   `eval` related commands (`gettitle`, `scrolltop`, etc.): Send JavaScript to be evaluated via the agent server's `/agents/{clientId}/eval` HTTP POST endpoint. Raw JavaScript is the POST body.
        *   `screenshot` command (New): Captures screenshots of specified DOM elements. It communicates with the agent server's `/agents/{clientId}/screenshot_elements` endpoint, sending selectors and receiving image data.
        *   Supports both fire-and-forget operations and waiting for results (`?wait=true` on relevant endpoints).
    *   **`vibrant calls` (`calls.go`)**: Commands to list and respond to tool calls, often from an AI interface, by running local tools (`../tools`) and injecting results back into the page.
    *   **`vibrant tools` (`tools.go`)**: Utilities to list, describe (JSON), and run local developer tools defined in the `../tools` package.
    *   **`vibrant canvas` (`canvas.go`)**: Starts a separate frontend web server (default `7777`) for UI and template rendering.

*   **Agent Web Service (`./web`)**: Implemented in `web/server.go`.
    *   **WebSocket Endpoint (`GET /agents/{clientId}/subscribe`)**: Chrome extension connects here.
    *   **Script Evaluation Endpoint (`POST /agents/{clientId}/eval`)**: Accepts raw JavaScript in the POST body. Wraps it in an `EVALUATE_SCRIPT` message and sends via WebSocket. Handles `?wait=true` for synchronous-like result fetching.
    *   **Element Screenshot Endpoint (`POST /agents/{clientId}/screenshot_elements`) (New)**: Accepts JSON `{"selectors": [...]}`. Sends a `CAPTURE_ELEMENTS_SCREENSHOT` message via WebSocket. Handles `?wait=true`.
    *   **Result Handling**: The WebSocket handler (`Conn.HandleMessage`) listens for `EVALUATION_RESULT` or `ELEMENTS_SCREENSHOT_RESULT` messages from the extension, correlates them using a `requestId`, and forwards results to any waiting HTTP handlers.

*   **Developer Tools (`./tools`)**: A framework for local file system tools like `read_file`, `list_files`, `create_file`, `apply_file_diff`.

### 2. Chrome DevTools Extension (`./plugins/chrome`)

*   **Core Function**: Acts as the execution bridge on the client-side.
*   **`manifest.json`**: Defines permissions including `storage`, `tabs`, `activeTab`, and `host_permissions` for `ws://localhost:9999/*` and crucially `<all_urls>` (needed for reliable `chrome.tabs.captureVisibleTab` operation).
*   **`background.js` (Service Worker)**: 
    *   Manages WebSocket connections to the agent server.
    *   Relays messages between `panel.js` and the agent server.
    *   Handles `REQUEST_TAB_CAPTURE` messages from the panel by calling `chrome.tabs.captureVisibleTab()` (using the inspected tab's `windowId`) and sending the resulting `dataUrl` back to the panel.
*   **`panel.js` (DevTools Panel "Agent Logger")**: 
    *   UI for connection management.
    *   Receives `EVALUATE_SCRIPT` messages: executes JS on the page using `chrome.devtools.inspectedWindow.eval()` and sends `EVALUATION_RESULT` back.
    *   Receives `CAPTURE_ELEMENTS_SCREENSHOT` messages (New):
        1.  Gets element bounding boxes via `eval`.
        2.  Requests full tab capture from `background.js`.
        3.  On receiving the full capture `dataUrl` from `background.js`, crops images for each selector using an in-memory canvas.
        4.  Sends `ELEMENTS_SCREENSHOT_RESULT` (with a map of `selector: dataURL`) back.

## Project Structure

*   **`./cmd`**: Go CLI (root, agents, client, tools, calls, etc.).
*   **`./web`**: Go backend for the agent WebSocket/HTTP server.
*   **`./tools`**: Go framework and implementations for local developer tools.
*   **`./plugins/chrome`**: Chrome DevTools extension.
*   **`./templates`**: Go HTML templates for `vibrant canvas`.
*   Other files: `go.mod`, `Makefile`, frontend build configs (`package.json`, `webpack.config.js`, etc.).

## How to Run (Core Agent Functionality)

1.  **Agent Server**: `go run . agents serve` (listens on `localhost:9999`).
2.  **Chrome Extension**: Load unpacked from `./plugins/chrome` in `chrome://extensions`. Reload and **close/reopen DevTools** after changes.
3.  **Interact**: 
    *   Open DevTools on a target page, open the "Agent Logger" panel, enter a client ID (e.g., "testClient"), and connect.
    *   Use `vibrant client -i testClient <subcommand>` (e.g., `gettitle`, `screenshot -s "h1" -o ./caps`).
