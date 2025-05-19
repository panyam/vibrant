# Project `vibrant` - Overall Summary

## Overview

`vibrant` is a Go-based toolchain augmented with a Chrome DevTools extension. Its primary goal is to facilitate AI-assisted coding and development by allowing a backend agent (controlled or monitored via the Go application) to interact with web pages being developed or inspected.

## Current State & Key Features

### 1. Go Application (`./`)

*   **CLI (`./cmd`)**: A command-line interface built with Cobra provides several top-level commands:
    *   `vibrant canvas`: Starts a web server (default port `7777`) using `panyam/templar` to serve a frontend application (likely for UI, testing, and dynamic templates).
    *   `vibrant agents serve`: Starts a dedicated HTTP/WebSocket server (default port `9999`) to handle communication with the Chrome DevTools extension. This server exposes:
        *   WebSocket endpoint: `ws://localhost:9999/agents/<connectionName>/subscribe` for the extension to connect.
        *   HTTP POST endpoints: `http://localhost:9999/agents/<connectionName>/commands/<COMMAND_TYPE>` for external triggers to send commands to connected extension instances.
    *   `vibrant tools`: A group of commands to list, describe in JSON, and run predefined developer tools.
*   **Agent Web Service (`./web`)**: Implements the backend logic for the `vibrant agents serve` command.
    *   Manages WebSocket connections from multiple Chrome DevTools extension instances.
    *   Receives commands via HTTP POST and broadcasts them over WebSockets to the targeted extension client(s).
    *   Handles specific command types by ensuring the correct JSON structure is relayed.
*   **Developer Tools (`./tools`)**: A framework for defining and running tools programmatically.
    *   Currently includes tools for file operations: `read_file`, `list_files`, `create_file`, `apply_file_diff`.
    *   These tools can be invoked via `vibrant tools run`.

### 2. Chrome DevTools Extension (`./plugins/chrome`)

*   **DevTools Panel**: Adds a custom panel ("Agent Logger") to Chrome DevTools.
*   **WebSocket Client**: Connects to the Go agent server (`ws://localhost:9999/agents/<connectionName>/subscribe`) via the panel UI.
    *   Includes auto-reconnection logic with exponential backoff if the connection drops (unless explicitly disconnected by the user).
*   **Page Interaction**: Receives commands (as JSON messages) from the Go agent server and executes corresponding actions on the inspected web page. This is achieved by dynamically generating and evaluating JavaScript code in the page's context using `chrome.devtools.inspectedWindow.eval()`.
*   **Supported Page Actions**:
    *   `SCROLL_TO_TOP`, `SCROLL_TO_BOTTOM`, `SCROLL_DELTA`: Perform page scrolling (with awareness of a potential custom `ms-autoscroll-container` element).
    *   `QUERY_SELECTOR_ALL`: Finds elements matching a CSS selector and returns their details (tag, id, class, rect, text, HTML).
    *   `SET_INPUT_VALUE`: Sets the value of input fields or textareas and dispatches `input` and `change` events. Can optionally trigger a click on a specified submit button.
    *   `CLICK_ELEMENT`: Dispatches a proper `MouseEvent` (click) on a selected element to trigger its event handlers.

## Project Structure

*   **`./cmd`**: Go CLI command definitions.
*   **`./web`**: Go package for the agent WebSocket/HTTP server.
*   **`./tools`**: Go package for defining and running developer tools.
*   **`./plugins/chrome`**: Chrome DevTools extension source files.
*   **`./templates`**: (Assumed) For HTML templates used by the `vibrant canvas` server.
*   **`./components`**: (Assumed) For TypeScript/TSX components for the frontend.
*   `go.mod`, `go.sum`: Go module files.
*   `Makefile`, `package.json`, `webpack.config.js`, `tailwind.config.js`: Build and frontend tooling.

## How to Run

1.  **Agent Server**: In one terminal, run `go run . agents serve` (defaults to port 9999).
2.  **Canvas/Frontend Server (Optional, if needed)**: In another terminal, run `go run . canvas` (defaults to port 7777).
3.  **Chrome Extension**: Load the unpacked extension from `./plugins/chrome` into Chrome via `chrome://extensions` (Developer mode enabled).
4.  **Interaction**: Open DevTools on a webpage, go to the "Agent Logger" panel, connect to an agent name. Then, trigger commands to the agent server (e.g., via its HTTP POST endpoints) to see actions performed on the inspected page.

## Context for New Developers

This project aims to bridge a backend Go environment with the browser's DevTools for enhanced development and debugging workflows, particularly with AI-driven interactions in mind. The Go backend can act as an "agent" that observes or manipulates a web page through the Chrome extension.

*   The **Go side** focuses on server logic, tool definitions, and CLI commands.
*   The **Chrome extension side** focuses on UI within DevTools, WebSocket communication, and securely evaluating JavaScript to interact with the target page.
*   Communication flow: HTTP POST (to Go) -> WebSocket (Go to Extension Background) -> Chrome Runtime Message (Extension Background to Panel) -> `inspectedWindow.eval()` (Panel to Page).
