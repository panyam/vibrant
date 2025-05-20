## Go Web Service Summary (`./web`)

This directory contains the Go package that implements the backend HTTP and WebSocket server for the `vibrant` agent system. It uses the `panyam/goutils/http` library for WebSocket connection management and `panyam/goutils/conc` for message fanout.

### Purpose

*   **WebSocket Server**: Provides the primary communication channel (`/agents/{clientId}/subscribe`) for Chrome DevTools extensions to connect.
*   **Remote Script Evaluation & Command Orchestration**: 
    *   Exposes an HTTP POST endpoint (`/agents/{clientId}/eval`) that accepts raw JavaScript code, packages it into an `EVALUATE_SCRIPT` WebSocket message (with a unique `requestId`), and broadcasts it to the specified connected Chrome extension (`clientId`).
    *   Exposes an HTTP POST endpoint (`POST /agents/{clientId}/screenshot_elements`) that accepts a list of CSS selectors, packages this into a `CAPTURE_ELEMENTS_SCREENSHOT` WebSocket message, and broadcasts it to the client.
    *   Optionally (`?wait=true` on the POST endpoints), waits for a corresponding `EVALUATION_RESULT` or `ELEMENTS_SCREENSHOT_RESULT` message back from the extension for the given `requestId`.
*   **Client & Request Management**: Manages active WebSocket connections per `clientId` and tracks pending requests that are awaiting a response.

### Key Components & Functionality (`server.go`)

1.  **`Request` Struct**:
    *   Represents an evaluation or command request sent to a client.
    *   Fields: `Id` (unique request ID), `ClientId`, `Payload` (original script, selectors, etc.), `SentAt`, `Response` (result from client), `ResponseAt`, `err`, `finished`.
    *   `recvChan chan string`: A buffered channel used by HTTP handlers (for `?wait=true` calls) to block and receive the script/command result from the WebSocket handler.

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
        *   If message `type` is `EVALUATION_RESULT`:
            *   Extracts `requestId`, `result`, `isException`, and `exceptionInfo`.
            *   Updates the corresponding pending `Request` and sends the result/error to `req.recvChan`.
        *   If message `type` is `ELEMENTS_SCREENSHOT_RESULT` (New):
            *   Extracts `requestId`, `imageData` (map of selector to dataURL), and `error`.
            *   Updates the corresponding pending `Request` and sends the `imageData` or error to `req.recvChan`.
        *   Marks the request as finished and removes it from `pendingRequests`.
    *   **`OnClose`**: Handles cleanup by removing the connection from the `FanOut`.

4.  **`Handler` Methods**:
    *   **`SubmitEvalRequest(clientId string, script string) *Request`**:
        *   Creates a `Request` for script evaluation, stores it, and broadcasts an `EVALUATE_SCRIPT` message.
    *   **`SubmitScreenshotElementsRequest(clientId string, selectors []string) *Request` (New)**:
        *   Creates a `Request` for element screenshots, stores it, and broadcasts a `CAPTURE_ELEMENTS_SCREENSHOT` message with the selectors.
    *   **`BroadcastToAgent(clientId string, payload map[string]any)`**: Sends the payload to the fanout for the specified `clientId`.

5.  **`NewServeMux()`**:
    *   Configures the HTTP routing for the agent server.
    *   `GET /agents/{clientId}/subscribe`: Handles WebSocket upgrade requests.
    *   `POST /agents/{clientId}/eval`: For script evaluations. Reads raw JS from body. Supports `?wait=true`.
    *   `POST /agents/{clientId}/screenshot_elements` (New): For capturing element screenshots. Expects `{"selectors": [...]}` in JSON body. Supports `?wait=true`.
    *   `GET /test_eval`: A simple GET endpoint for testing script evaluation.

### Workflow Summary (Illustrative for Screenshots)

1.  Agent server (`vibrant agents serve`) listens (default `localhost:9999`).
2.  Chrome DevTools extension connects to `GET /agents/{clientId}/subscribe`.
3.  CLI (`vibrant client screenshot ...`) makes a `POST /agents/{clientId}/screenshot_elements?wait=true` request with selectors in the JSON body.
4.  The HTTP handler calls `handler.SubmitScreenshotElementsRequest`. This generates a `requestId`, stores the request, and broadcasts a `CAPTURE_ELEMENTS_SCREENSHOT` message over WebSocket.
5.  The Chrome extension receives this message, captures/crops element images, and sends an `ELEMENTS_SCREENSHOT_RESULT` message (with `requestId`, `imageData` map, and any error) back over WebSocket.
6.  `Conn.HandleMessage` on the server receives this result, finds the pending `Request`, and sends the `imageData` map (or error) to the `recvChan` of that `Request`.
7.  The original HTTP handler for `/screenshot_elements` (which was waiting due to `?wait=true`) receives the data from `recvChan` and includes it in its HTTP response to the CLI.
