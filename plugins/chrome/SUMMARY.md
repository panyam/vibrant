## Chrome DevTools Extension Summary (`./plugins/chrome`)

This directory contains the Chrome DevTools extension that acts as the client-side agent for `vibrant`. It connects to the Go backend agent server via WebSockets and executes commands or performs actions requested by the backend in the context of the inspected page.

### Purpose

*   Establish a WebSocket connection to the `vibrant` agent server.
*   Listen for commands from the agent server, primarily:
    *   `EVALUATE_SCRIPT`: Execute arbitrary JavaScript in the inspected page and return the result.
    *   `CAPTURE_ELEMENTS_SCREENSHOT`: Capture images of specified DOM elements and return the image data.
*   Send results or data back to the agent server.
*   Provide a DevTools panel ("Agent Logger") for connection management and status visibility.

### Key Components & Functionality

1.  **`manifest.json`**:
    *   Defines the extension's properties, permissions, and entry points.
    *   **Permissions**: `storage`, `tabs`, `activeTab`.
    *   **Host Permissions**: `ws://localhost:9999/*`, `wss://localhost:9999/*`, and critically `<all_urls>` (required for `chrome.tabs.captureVisibleTab()` to function reliably in this extension's operational context).
    *   Specifies `devtools.html` as the DevTools page and `background.js` as the service worker.

2.  **`devtools.html` & `devtools.js`**:
    *   Basic HTML page that loads `devtools.js`.
    *   `devtools.js` creates the "Agent Logger" panel in Chrome DevTools, pointing it to `panel.html`.

3.  **`background.js` (Service Worker)**:
    *   Manages WebSocket connections for each DevTools panel instance.
    *   Relays messages between the panel (`panel.js`) and the WebSocket server.
    *   Handles `CONNECT_WEBSOCKET` and `DISCONNECT_WEBSOCKET` messages from the panel.
    *   Forwards messages of type `FORWARD_TO_WEBSOCKET_SERVER` (which contain payloads like `EVALUATION_RESULT` or `ELEMENTS_SCREENSHOT_RESULT`) from the panel to the Go agent server via WebSocket.
    *   **Handles `REQUEST_TAB_CAPTURE` messages from `panel.js` (New)**:
        *   Receives the `tabId` of the inspected window and a `requestId`.
        *   Calls `chrome.tabs.get(tabId, ...)` to get the `windowId` of the inspected tab.
        *   Calls `chrome.tabs.captureVisibleTab(windowId, { format: "png" }, ...)` to capture the active tab in that window.
        *   Sends a `TAB_CAPTURE_COMPLETE` message back to `panel.js` containing the `dataUrl` of the screenshot (or an error) and the original `requestId`.
    *   Manages cleanup when tabs are removed or updated.

4.  **`panel.html` & `panel.js`**:
    *   `panel.html`: Provides the UI for the DevTools panel (connection input, connect/disconnect buttons, status display). Styled to respect Chrome DevTools themes.
    *   `panel.js`: Contains the core client-side logic.
        *   Manages the connection lifecycle to the background script and, through it, to the WebSocket server.
        *   Includes auto-reconnection logic for the WebSocket connection.
        *   **Message Handling (from background, originating from server)**:
            *   `EVALUATE_SCRIPT`: Calls `handleEvaluateScriptRequest`.
            *   `CAPTURE_ELEMENTS_SCREENSHOT`: Calls `handleCaptureElementsScreenshotRequest` (New).
        *   **`handleEvaluateScriptRequest(requestDetails)`**:
            *   Uses `chrome.devtools.inspectedWindow.eval()` to execute the provided JavaScript.
            *   Sends an `EVALUATION_RESULT` message back to `background.js` for forwarding.
        *   **`handleCaptureElementsScreenshotRequest(requestDetails)` (New)**:
            1.  Receives selectors and a `requestId`.
            2.  Executes a script in the inspected page via `chrome.devtools.inspectedWindow.eval()` to get `getBoundingClientRect()` and `devicePixelRatio` for each selector.
            3.  Sends a `REQUEST_TAB_CAPTURE` message to `background.js` with the `inspectedWindow.tabId` and `requestId`.
            4.  Awaits a `TAB_CAPTURE_COMPLETE` message from `background.js` (this message contains the full-page `dataUrl` or an error).
            5.  If successful, loads the full-page `dataUrl` into an `Image` object.
            6.  Uses an in-memory `<canvas>` to crop sections of the main image corresponding to each element's bounding box (adjusted for DPR).
            7.  Converts each cropped canvas to a PNG data URL.
            8.  Sends an `ELEMENTS_SCREENSHOT_RESULT` message (containing a map of `selector: dataURL` and the `requestId`) back to `background.js` for forwarding to the Go server.
        *   **Message Handling (from background, for screenshot step)**:
            *   `TAB_CAPTURE_COMPLETE`: Receives the full-page screenshot data (or error) from `background.js` and resolves/rejects a promise that `handleCaptureElementsScreenshotRequest` is awaiting.

### Screenshot Workflow within Extension

1.  Go Backend sends `CAPTURE_ELEMENTS_SCREENSHOT` (with selectors, requestId) -> `panel.js`.
2.  `panel.js` gets element bounding boxes from the page via `eval`.
3.  `panel.js` sends `REQUEST_TAB_CAPTURE` (with tabId, requestId) -> `background.js`.
4.  `background.js` calls `chrome.tabs.captureVisibleTab()` using the window ID of the inspected tab.
5.  `background.js` sends `TAB_CAPTURE_COMPLETE` (with full page dataURL, requestId) -> `panel.js`.
6.  `panel.js` receives dataURL, crops images for each selector using a canvas.
7.  `panel.js` sends `ELEMENTS_SCREENSHOT_RESULT` (with map of selector:croppedDataURL, requestId) -> Go Backend.
