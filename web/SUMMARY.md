## Go Web Service Summary (`./web`)

This directory contains the Go package that implements the backend HTTP and WebSocket server for the `vibrant` agent system. It uses the `panyam/goutils/http` library for WebSocket connection management and `panyam/goutils/conc` for message fanout.

### Purpose

*   **WebSocket Server**: Provides the primary communication channel (`/agents/{clientId}/subscribe`) for Chrome DevTools extensions to connect.
*   **Remote Script Evaluation & Command Orchestration**: 
    *   Exposes an HTTP POST endpoint (`/agents/{clientId}/eval`) that accepts raw JavaScript code, packages it into an `EVALUATE_SCRIPT` WebSocket message (with a unique `requestId`), and broadcasts it to the specified connected Chrome extension (`clientId`).
    *   Exposes an HTTP POST endpoint (`POST /agents/{clientId}/screenshots`) that accepts a list of CSS selectors, packages this into a `CAPTURE_ELEMENTS_SCREENSHOT` WebSocket message, and broadcasts it to the client.
    *   Exposes an HTTP POST endpoint (`POST /agents/{clientId}/paste`) that accepts a selector and data URL, packages this into a `PASTE_DATA` WebSocket message, and broadcasts it to the client. (New)
    *   Optionally (`?wait=true` on POST endpoints), waits for a corresponding result message (e.g., `EVALUATION_RESULT`, `ELEMENTS_SCREENSHOT_RESULT`, `PASTE_RESULT`) back from the extension for the given `requestId`.
*   **Client & Request Management**: Manages active WebSocket connections per `clientId` and tracks pending requests that are awaiting a response.

### Key Components & Functionality (`server.go`)

1.  **`Request` Struct**:
    *   Represents a command request sent to a client.
    *   Fields: `Id` (unique request ID), `ClientId`, `Payload` (original script, selectors, or paste data), `SentAt`, `Response` (result from client), `ResponseAt`, `err`, `finished`.
    *   `recvChan chan string`: A buffered channel used by HTTP handlers (for `?wait=true` calls) to block and receive the command result from the WebSocket handler.

2.  **`Handler` Struct**:
    *   Centralizes connection and request management.
    *   `Fanouts map[string]*conc.FanOut[conc.Message[any]]`: Manages broadcasting messages to all WebSocket connections for a given `clientId`.
    *   `pendingRequests map[string]*Request`: Stores `Request` objects keyed by their `Id`, used to correlate incoming results with the originating HTTP request if it's waiting.

3.  **`Conn` Struct (implements `gohttp.WSHandler`)**:
    *   Wraps `gohttp.JSONConn` to handle individual WebSocket connections.
    *   **`Validate`**: Extracts `clientId` from the path for new WebSocket connections.
    *   **`OnStart`**: Called when a new WebSocket connection is established. Registers the connection with the `FanOut` and sends a welcome script.
    *   **`HandleMessage`**: Crucial for receiving results from the client.
        *   Parses incoming JSON messages from the WebSocket client.
        *   Handles `EVALUATION_RESULT`, `ELEMENTS_SCREENSHOT_RESULT`, and `PASTE_RESULT` (New) message types.
        *   For each, extracts `requestId` and relevant data (result, imageData, success/error/message for paste).
        *   Updates the corresponding pending `Request` and sends the structured result (as a JSON string) to `req.recvChan`.
        *   Marks the request as finished and removes it from `pendingRequests`.
    *   **`OnClose`**: Handles cleanup by removing the connection from the `FanOut`.

4.  **`Handler` Methods**:
    *   **`SubmitRequest(reqType string, req *Request)`**: (Generalized method)
        *   Stores the `Request` in `pendingRequests`.
        *   Constructs a WebSocket message with `type`, `requestId`, and `payload`.
        *   Broadcasts the message to the appropriate agent using `BroadcastToAgent`.

5.  **`NewServeMux()`**:
    *   Configures the HTTP routing for the agent server.
    *   `GET /agents/{clientId}/subscribe`: Handles WebSocket upgrade requests.
    *   `POST /agents/{clientId}/eval`: For script evaluations. Supports `?wait=true`.
    *   `POST /agents/{clientId}/screenshots`: For capturing element screenshots. Expects `{"selectors": [...]}`. Supports `?wait=true`.
    *   `POST /agents/{clientId}/paste` (New): For pasting data. Expects `{"selector": "...", "dataUrl": "..."}`. Supports `?wait=true`.
    *   API responses for `?wait=true` calls are structured as `{"requestId": "...", "response": <actual_result_or_error_object>}`.

### Workflow Summary (Illustrative for Paste Command - New)

1.  Agent server (`vibrant agents serve`) listens (default `localhost:9999`).
2.  Chrome DevTools extension connects to `GET /agents/{clientId}/subscribe`.
3.  CLI (`vibrant client paste ...`) makes a `POST /agents/{clientId}/paste?wait=true` request with `{"selector": "...", "dataUrl": "..."}` in the JSON body.
4.  The HTTP handler for `/paste` calls `handler.SubmitRequest` with type `PASTE_DATA`. This generates a `requestId`, stores the request, and broadcasts a `PASTE_DATA` message (payload includes selector and dataUrl) over WebSocket to the extension.
5.  The Chrome extension (`panel.js`) receives this message, executes a script in the inspected page to simulate the paste event using the provided dataUrl on the target selector.
6.  The extension sends a `PASTE_RESULT` message (with `requestId`, `success` status, and `message`/`error`) back over WebSocket.
7.  `Conn.HandleMessage` on the server receives this result, finds the pending `Request`, and sends the `PASTE_RESULT`'s payload (as a JSON string) to the `recvChan` of that `Request`.
8.  The original HTTP handler for `/paste` (which was waiting due to `?wait=true`) receives the data from `recvChan` and includes it in its HTTP response to the CLI, structured as `{"requestId": "...", "response": {"success": ..., "message": ..., "error": ...}}`.
