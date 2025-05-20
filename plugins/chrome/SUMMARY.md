## Chrome DevTools Extension Summary (`./plugins/chrome`)

This directory contains the Chrome DevTools extension that acts as the client-side agent for `vibrant`. It connects to the Go backend agent server via WebSockets and executes commands or performs actions requested by the backend in the context of the inspected page.

### Purpose

*   Establish a WebSocket connection to the `vibrant` agent server.
*   Listen for commands from the agent server, primarily:
    *   `EVALUATE_SCRIPT`: Execute arbitrary JavaScript in the inspected page and return the result.
    *   `CAPTURE_ELEMENTS_SCREENSHOT`: Capture images of specified DOM elements and return the image data.
    *   `PASTE_DATA` (New): Simulate a paste event on a target element with provided data (e.g., image data URL).
*   Send results or data back to the agent server.
*   Provide a DevTools panel ("Agent Logger") for connection management and status visibility.

### Key Components & Functionality

1.  **`manifest.json`**:
    *   Defines the extension's properties, permissions, and entry points.
    *   **Permissions**: `storage`, `tabs`, `activeTab`, `clipboardRead`, `clipboardWrite` (the clipboard permissions are important for robust paste simulation, as some web apps might try to interact with `navigator.clipboard` during their paste handlers).
    *   **Host Permissions**: `ws://localhost:9999/*`, `wss://localhost:9999/*`, and `<all_urls>`.
    *   Specifies `devtools.html` as the DevTools page and `background.js` as the service worker.

2.  **`devtools.html` & `devtools.js`**:
    *   Standard DevTools extension setup files to create the "Agent Logger" panel.

3.  **`background.js` (Service Worker)**:
    *   Manages WebSocket connections for each DevTools panel instance.
    *   Relays messages between `panel.js` and the agent server.
    *   Handles `REQUEST_TAB_CAPTURE` messages from `panel.js` for the screenshot functionality.

4.  **`panel.html` & `panel.js`**:
    *   `panel.html`: UI for the DevTools panel.
    *   `panel.js`: Core client-side logic.
        *   Manages WebSocket connection and reconnection.
        *   **Message Handling (from agent server via background.js)**:
            *   `EVALUATE_SCRIPT`: Calls `handleEvaluateScriptRequest`.
            *   `CAPTURE_ELEMENTS_SCREENSHOT`: Calls `handleCaptureElementsScreenshotRequest`.
            *   `PASTE_DATA` (New): Calls `handlePasteDataRequest`.
        *   **`handleEvaluateScriptRequest(requestDetails)`**:
            *   Uses `chrome.devtools.inspectedWindow.eval()` to execute JavaScript.
            *   Sends `EVALUATION_RESULT` back (with `result`, `isException`, `exceptionInfo`).
        *   **`handleCaptureElementsScreenshotRequest(requestDetails)`**:
            *   Gets element bounding boxes, requests full tab capture from `background.js`, crops images using a canvas, and sends `ELEMENTS_SCREENSHOT_RESULT` (with `imageData` map or `error`).
        *   **`handlePasteDataRequest(requestDetails)` (New)**:
            1.  Receives `selector`, `dataUrl`, and `requestId`.
            2.  Constructs and executes a JavaScript snippet via `chrome.devtools.inspectedWindow.eval()`.
            3.  The script in the inspected page:
                *   Finds the target element by `selector`.
                *   Focuses the element.
                *   Fetches the `dataUrl` to get a `Blob`.
                *   Creates a `DataTransfer` object and adds the `Blob` as a `File`.
                *   Dispatches a synthetic `ClipboardEvent` of type `'paste'` on the target element, with the `clipboardData` property set to the created `DataTransfer` object.
                *   Returns `{success: true/false, message: string, error: string}`.
            4.  Sends a `PASTE_RESULT` message back to the agent server (with `requestId`, `success`, `message`, `error`).

### Paste Workflow within Extension (New)

1.  Go Backend sends `PASTE_DATA` (with `selector`, `dataUrl`, `requestId`) -> `panel.js`.
2.  `panel.js` calls `handlePasteDataRequest`.
3.  `handlePasteDataRequest` prepares a JavaScript function call (as a string) to be executed in the inspected page.
4.  This script is run using `chrome.devtools.inspectedWindow.eval()`.
5.  The script on the inspected page finds the element, fetches the data from `dataUrl` into a Blob, creates a `File` and `DataTransfer` object, and dispatches a `paste` event on the element with the `DataTransfer` object.
6.  The script returns a success/failure status.
7.  `panel.js` receives this status from the `eval` callback and sends a `PASTE_RESULT` (with the status and `requestId`) back to the Go Backend.
