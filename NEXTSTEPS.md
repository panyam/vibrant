# Next Steps & Outstanding Work for `vibrant`

This document outlines potential areas for future development, improvement, and features for the `vibrant` project.

## I. Core Agent & Extension Enhancements

1.  **Bi-directional Communication (Extension to Agent)**:
    *   **Problem**: Currently, the Chrome extension primarily receives commands. Results from `QUERY_SELECTOR_ALL` are logged but not systematically sent back to the Go agent.
    *   **Next Step**: Implement a mechanism for `panel.js` (or `background.js`) to send messages (e.g., query results, status updates from page interactions) back to the Go WebSocket server.
    *   The Go server (`web/server.go`) would need to handle messages received on the WebSocket connection from clients.

2.  **More Sophisticated Page Interactions/Queries**:
    *   **Problem**: Current interactions are basic. More complex queries or manipulations might be needed.
    *   **Next Steps**:
        *   Add commands to get/set element attributes.
        *   Add command to get computed styles of an element.
        *   Allow execution of arbitrary (but safe) JavaScript snippets sent from the agent and return results.
        *   Support for interacting with iframes within the inspected page.

3.  **Response Handling & Correlation**:
    *   **Problem**: If the agent sends multiple commands (especially queries), responses need to be correlated to requests.
    *   **Next Step**: Ensure all relevant commands (like `QUERY_SELECTOR_ALL`) include a unique `requestId`. The response sent back from the extension should include this `requestId`.

4.  **Error Handling & Reporting**: 
    *   **Problem**: Error handling is present but could be more robust and user-friendly.
    *   **Next Steps**:
        *   Standardize error object formats sent over WebSocket.
        *   Display errors more clearly in the DevTools panel UI, perhaps with codes or more context.
        *   Improve error logging and diagnostics on both Go server and extension sides.

5.  **Security for `inspectedWindow.eval()`**: 
    *   **Problem**: While powerful, `eval` can be a risk if the agent server could be compromised or if commands are not carefully constructed.
    *   **Next Step**: Review and potentially sanitize or validate inputs to `evalString` construction in `panel.js`, although the primary trust boundary is the agent server itself.

## II. Go Application Improvements

1.  **Agent Server (`./web`)**:
    *   **Command Authentication/Authorization**: Secure the POST command endpoints (`/agents/...`) if the agent server is exposed beyond localhost or in a multi-user environment.
    *   **Configuration**: Make WebSocket server settings (e.g., buffer sizes, allowed origins if not just localhost) configurable.
    *   **Connection Name Management**: Consider if `connectionName` needs more robust management (e.g., ensuring uniqueness, session handling if agents are long-lived).

2.  **CLI (`./cmd`)**:
    *   **Parameter Input for `vibrant tools run`**: Reading JSON from stdin is functional but not very user-friendly. Explore alternative ways to pass complex parameters (e.g., individual flags, reading from a file).
    *   **Configuration Files**: Allow parts of the application (e.g., default ports, tool project roots) to be configured via a config file (`.vibrant.yaml` or similar).

3.  **Tools Framework (`./tools`)**:
    *   **`list_files` Recursion**: Properly implement the `recurse` parameter in the `ListFiles` tool.
    *   **`apply_file_diff` Robustness**: The `patch` command can be brittle. Explore Go-native diff/patch libraries for more control and better error reporting.
    *   **New Tool Types**: Consider tools for interacting with other development services, version control, etc.

## III. Chrome Extension UI/UX

1.  **DevTools Panel (`panel.html`, `panel.js`)**:
    *   **Connection Status**: Provide more granular connection status (e.g., "Connecting to background script...", "Background connected, connecting WebSocket...").
    *   **Message Log in Panel**: Display a log of messages/commands sent and received directly within the panel UI, not just in the console.
    *   **Command History/Replay**: Allow users to see a history of commands sent and potentially resend them.
    *   **Configuration**: Allow some settings (e.g., default connection name, verbosity) to be configured via the panel UI (using `chrome.storage`).

## IV. Build & Development Workflow

1.  **Frontend Build (`webpack.config.js`, `package.json`)**:
    *   Review and optimize the frontend build process if it becomes slow or complex.
    *   Ensure dependencies are up-to-date.
2.  **Testing**: 
    *   Implement unit tests for Go packages (`web`, `tools`, `cmd`).
    *   Explore options for testing the Chrome extension (e.g., using Puppeteer or similar browser automation tools for integration tests).

## V. Documentation

1.  **User Guide**: Create a more detailed user guide for setting up and using the `vibrant` CLI and the Chrome extension.
2.  **Developer Guide**: Expand on API contracts (WebSocket message formats, HTTP endpoint specs) for developers wanting to integrate or extend the system.

## Suggested Starting Points for Next Steps

1.  **Bi-directional Communication**: Focus on sending results from `QUERY_SELECTOR_ALL` back to the Go agent. This will involve modifying `panel.js` to send a message, `background.js` to relay it, and `web/server.go` to handle incoming WebSocket messages.
2.  **Message Log in Panel UI**: This would significantly improve the usability of the DevTools extension for debugging.
3.  **Refine `vibrant tools run` Parameter Input**: Improve CLI usability for tool execution.
