# Next Steps & Outstanding Work for `vibrant`

This document outlines potential areas for future development, improvements, and features based on the current architecture.

## I. Core Remote Evaluation & Extension Enhancements

1.  **Complex Result Handling from `EVALUATE_SCRIPT`**:
    *   **Status**: Currently, simple results are handled. For very complex, non-serializable objects (like direct DOM element references), the evaluated script itself should serialize its result (e.g., `JSON.stringify(details)`). The Go backend might need more robust unmarshalling for diverse expected types if scripts return complex JSON strings.
    *   **Next Step**: Review if current handling is sufficient or if specific commands require more sophisticated serialization/deserialization strategies on both client and server.

2.  **Streaming Large Results or Continuous Data**:
    *   **Problem**: The current request/response model is for single interactions. Streaming logs or continuous data from the page is not directly supported by one command.
    *   **Next Step**: For true streaming, the Go backend could issue many small `EVALUATE_SCRIPT` calls to poll for data, or a more complex system involving the page script setting up listeners and initiating messages (perhaps via `console.log` scraping or a dedicated "send message" `EVALUATE_SCRIPT` call) could be designed if needed.

3.  **Advanced Screenshot Features**:
    *   **Full Page Screenshots (Beyond Viewport)**: The current `captureVisibleTab` only gets the viewport. True full-page screenshots (including content not scrolled into view) are significantly more complex and usually require page stitching or browser-internal APIs (like Puppeteer's full page screenshot, not directly usable in an extension this way).
    *   **Scrolling Elements into View Before Capture**: For `CAPTURE_ELEMENTS_SCREENSHOT`, if an element is outside the viewport, its capture will be blank/incorrect. The Go CLI/backend could be enhanced to optionally send a script to scroll an element into view before requesting its screenshot.
    *   **Different Image Formats/Quality**: Allow CLI to specify format (JPEG) or quality for screenshots.

4.  **Error Propagation and Display**:
    *   **CLI**: Improve user-facing error messages in the CLI when `sendEvalScript` or screenshot commands fail, detailing issues from the agent or extension.
    *   **Panel**: The panel logs to its console, but for user-initiated actions from the CLI, there's no direct UI feedback in the panel. This is generally fine as the CLI provides feedback.

## II. Go Application (Backend & CLI)

1.  **State Management in Go Agent**:
    *   If the Go agent needs to make more complex decisions based on a sequence of interactions, a more formal state management approach for each `clientId` might be needed.

2.  **CLI Enhancements (`cmd/client.go`, etc.)**:
    *   **More Commands**: Add more `vibrant client` subcommands for common page interactions (e.g., click element, check visibility, get attributes, fill forms more generically).
    *   **Output Formatting**: Improve CLI output formatting for diverse results (JSON pretty-printing, tables, etc.).
    *   **File Uploads/Downloads via Agent**: A more advanced feature could involve proxying file interactions if the agent needs to get or put files onto the inspected page's domain (requires careful security considerations).

3.  **Security for Agent Server**:
    *   **Status**: Currently listens on `localhost`. If exposed externally, authentication/authorization for WebSocket connections and HTTP endpoints would be critical.
    *   **Next Step**: Implement API keys or other auth mechanisms if the server needs to be accessed non-locally.

4.  **Robustness of Pending Request Cleanup (`web/server.go`)**:
    *   **Status**: Timeouts are implemented for waiting HTTP requests. Ensure that `pendingRequests` map entries are always cleaned up, even for non-waiting requests if the client disconnects abruptly before sending a result.
    *   **Next Step**: Review `OnClose` logic for WebSocket connections in `web/server.go` to ensure it actively clears any associated `pendingRequests` that might not have a waiting HTTP handler with its own timeout.

## III. Chrome Extension UI/UX (Panel)

1.  **Panel Activity Log (Low Priority for CLI-driven tool)**:
    *   Consider a small, unobtrusive log within the panel for recent commands and results for direct debugging of the panel itself. Current console logging is good for development.

2.  **Manual Script Input in Panel (Advanced/Optional)**:
    *   Could add a textarea in the panel for developers to manually type and send JavaScript via the `EVALUATE_SCRIPT` pathway for quick tests, without needing the CLI.

## IV. Testing & Build

1.  **Go Unit/Integration Tests**: Write more comprehensive tests for the `web` package (request/response lifecycles, WebSocket message handling) and for CLI command logic in `cmd`.
2.  **Extension Testing**: Manual testing is primary. Consider basic automated tests if the extension UI becomes more complex.
3.  **Build Process for Extension**: Currently manual load. For easier distribution or versioning, a simple build script (e.g., zip creation) could be added to the Makefile.

## Suggested Starting Points for Immediate Next Steps

1.  **Refine Screenshot Error Handling/Reporting**: Ensure errors from any stage of the screenshot process (element not found, capture fail, crop fail) are clearly reported back to the CLI user.
2.  **Scroll Element into View before Screenshot**: Add an option or a separate command (`vibrant client scroll-to-selector <selector>`) that the user can invoke before `screenshot` if an element might be off-screen.
3.  **Expand `vibrant client` Commands**: Add a few more common interaction commands (e.g., `click <selector>`, `get-attribute <selector> <attributeName>`).
4.  **Review `pendingRequests` Cleanup**: Double-check the logic in `web/server.go`'s `OnClose` to ensure all orphaned pending requests for a disconnected client are cleared to prevent potential memory leaks if many fire-and-forget commands were issued without responses.
