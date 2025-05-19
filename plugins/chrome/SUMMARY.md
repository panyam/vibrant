## Chrome DevTools Extension Summary (`./plugins/chrome`)

This directory contains a Chrome DevTools extension designed to facilitate communication between a backend Go agent and an inspected web page.

### Purpose

*   **Agent Communication**: Establishes a WebSocket connection to a backend Go agent server.
*   **Page Interaction**: Receives commands from the agent server and executes them on the currently inspected web page. This allows the backend to programmatically control and query the web page.
*   **Developer Feedback**: Logs actions and results to the DevTools console and the inspected page's console.

### Key Components & Functionality

1.  **`manifest.json`**:
    *   Defines the extension (version 3).
    *   Declares `devtools_page` (`devtools.html`) as the entry point for DevTools integration.
    *   Specifies necessary `permissions` (storage, activeTab) and `host_permissions` (for `ws://localhost:9999/*`).
    *   Registers the `background.js` service worker.
    *   Defines icons.

2.  **`devtools.html` & `devtools.js`**:
    *   `devtools.html` is a minimal HTML page that loads `devtools.js`.
    *   `devtools.js` uses `chrome.devtools.panels.create()` to add a custom panel named "Agent Logger" to the Chrome DevTools window. This panel's UI is defined by `panel.html`.

3.  **`panel.html` & `panel.js` (The User Interface & Core Client Logic)**:
    *   **UI (`panel.html`)**: Provides a simple interface within the "Agent Logger" DevTools panel:
        *   An input field for the "Connection Name".
        *   "Connect" and "Disconnect" buttons.
        *   A status display area.
    *   **Logic (`panel.js`)**:
        *   **Connection Management**:
            *   Establishes a long-lived communication port with `background.js`.
            *   Handles user clicks on "Connect" and "Disconnect" buttons.
            *   Sends `CONNECT_WEBSOCKET` and `DISCONNECT_WEBSOCKET` messages to `background.js`.
            *   Stores the last used connection name in `sessionStorage` for convenience.
        *   **Auto-Reconnection**:
            *   If the WebSocket connection drops (and it wasn't a user-initiated disconnect), it attempts to reconnect automatically using an exponential backoff strategy (initial delay 1s, max delay 10s).
            *   The user can always manually click "Connect" to override/initiate a connection.
        *   **Message Handling**:
            *   Receives `WEBSOCKET_MESSAGE` (containing commands from the Go server) and `WEBSOCKET_STATUS` updates from `background.js`.
            *   Calls `executeInInspectedPage()` to process commands.
        *   **`executeInInspectedPage(commandDetails)`**: This is the core of page interaction.
            *   It takes the command data (parsed JSON from the WebSocket message).
            *   Uses a `switch` statement based on `commandDetails.type`.
            *   Constructs JavaScript code (`evalString`) to be executed in the context of the inspected page using `chrome.devtools.inspectedWindow.eval()`.
            *   **Supported Commands**:
                *   `SCROLL_TO_TOP`: Scrolls the page (or `ms-autoscroll-container` if present) to the top.
                *   `SCROLL_TO_BOTTOM`: Scrolls the page (or `ms-autoscroll-container`) to the bottom.
                *   `SCROLL_DELTA`: Scrolls by a specified `deltaX` and `deltaY`.
                *   `QUERY_SELECTOR_ALL`: Executes `document.querySelectorAll()` with the given selector, extracts details (tagName, id, className, rect, innerText, outerHTML) for each element, and logs the results.
                *   `SET_INPUT_VALUE`: Sets the `.value` of a target input/textarea element.
                    *   Dispatches `input` and `change` events to simulate user interaction and trigger page listeners.
                    *   Optionally, if `submit: true` and `submitSelector` are provided in the command, it will also dispatch a click event on the specified submit button.
                *   `CLICK_ELEMENT`: Finds an element by selector and dispatches a `MouseEvent` ('click') to it, ensuring event listeners are triggered.
            *   Logs actions, results, and errors to both the panel's console and the inspected page's console using distinct prefixes (`[AgentAction]`, `[AgentQueryResult]`, etc.).

4.  **`background.js` (Service Worker)**:
    *   **Persistent Logic**: Manages WebSocket connections, as panel UI can be closed/reopened.
    *   **Connection Multiplexing**: Maintains a map of active WebSocket connections, keyed by `tabId`, allowing multiple DevTools panels (for different tabs) to have independent agent connections.
    *   **WebSocket Handling**:
        *   Receives "CONNECT_WEBSOCKET" messages from `panel.js`.
        *   Establishes WebSocket connection to `ws://localhost:9999/agents/<connectionName>/subscribe`.
        *   Relays messages received from the WebSocket server to the appropriate `panel.js` instance (for the correct tab).
        *   Sends status updates (`WEBSOCKET_STATUS`) back to `panel.js` (e.g., "Connected", "Disconnected", "Error").
    *   **Lifecycle Management**:
        *   Listens for `chrome.tabs.onRemoved` and `chrome.tabs.onUpdated` (for navigation) to clean up and close WebSockets associated with closed or navigated-away tabs.
        *   Handles `onDisconnect` events from the panel's communication port.

### Workflow Summary

1.  User opens DevTools on a webpage.
2.  The "Agent Logger" panel is available.
3.  User enters a "Connection Name" and clicks "Connect".
4.  `panel.js` tells `background.js` to establish a WebSocket connection for that tab and name.
5.  `background.js` connects to `ws://localhost:9999/agents/<name>/subscribe`.
6.  Status is relayed back to `panel.js`.
7.  When the Go agent server sends a JSON message over WebSocket, `background.js` forwards it to the relevant `panel.js`.
8.  `panel.js`'s `executeInInspectedPage` function interprets the command in the message and uses `chrome.devtools.inspectedWindow.eval()` to run JavaScript code on the inspected page, performing the requested action (scroll, query, set value, click).
9.  If the WebSocket connection drops, `panel.js` attempts to reconnect automatically unless the user explicitly disconnected.
