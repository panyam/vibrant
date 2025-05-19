## Go Web Service Summary (`./web`)

This directory contains the Go package that implements the backend HTTP and WebSocket server for the `vibrant` agent system. It uses the `panyam/goutils/http` library for WebSocket connection management and `panyam/goutils/conc` for message fanout.

### Purpose

*   **WebSocket Server**: Provides the primary communication channel (`/agents/{clientId}/subscribe`) for Chrome DevTools extensions to connect.
*   **Remote Script Evaluation Orchestration**:
    *   Exposes an HTTP POST endpoint (`/agents/{clientId}/eval`) that accepts raw JavaScript code.
    *   Packages this script into an `EVALUATE_SCRIPT` WebSocket message (with a unique `requestId`).
    *   Broadcasts this message to the specified connected Chrome extension (`clientId`).
    *   Optionally (`?wait=true` on the POST endpoint), waits for an `EVALUATION_RESULT` message back from the extension for the given `requestId`.
*   **Client & Request Management**: Manages active WebSocket connections per `clientId` and tracks pending evaluation requests that are awaiting a response.

### Key Components & Functionality (`server.go`)

1.  **`Request` Struct**:
    *   Represents an evaluation request sent to a client.
    *   Fields: `Id` (unique request ID), `ClientId`, `Payload` (original script), `SentAt`, `Response` (result from client), `ResponseAt`, `err`, `finished`.
    *   `recvChan chan string`: A buffered channel used by the `/eval?wait=true` HTTP handler to block and receive the script evaluation result from the WebSocket handler.

2.  **`Handler` Struct**:
    *   Centralizes connection and request management.
    *   `Fanouts map[string]*conc.FanOut[conc.Message[any]]`: Manages broadcasting messages to all WebSocket connections for a given `clientId`. Uses `panyam/goutils/conc.FanOut`.
    *   `pendingRequests map[string]*Request`: Stores `Request` objects keyed by their `Id`, used to correlate incoming `EVALUATION_RESULT` messages with the originating HTTP request if it's waiting.

3.  **`Conn` Struct (implements `gohttp.WSHandler`)**:
    *   Wraps `gohttp.JSONConn` to handle individual WebSocket connections.
    *   **`Validate`**: Extracts `clientId` from the path for new WebSocket connections.
    *   **`OnStart`**: Called when a new WebSocket connection is established.
        *   Registers the connection with the appropriate `FanOut` in the `Handler`.
        *   Proactively sends a "welcome" `EVALUATE_SCRIPT` message to the newly connected client (e.g., to log a welcome message in the client's console and get basic page info).
    *   **`HandleMessage`**: **Crucial for receiving results.**
        *   Called when a message (expected to be JSON) is received from the WebSocket client.
        *   Parses the message and checks if its `type` is `EVALUATION_RESULT`.
        *   If so, it extracts the `requestId`, `result`, `isException`, and `exceptionInfo`.
        *   Looks up the original `Request` in `handler.pendingRequests` using the `requestId`.
        *   If found and not already finished, it updates the `Request` object with the response details and sends the serialized result (or error) to the `req.recvChan`.
        *   Marks the request as finished and removes it from `pendingRequests`.
    *   **`OnClose`**: Handles cleanup by removing the connection from the `FanOut`.
    *   **`OnTimeout`**: Logs timeout.

4.  **`Handler` Methods**:
    *   **`SubmitEvalRequest(clientId string, script string) *Request`**:
        *   Creates a `Request` object.
        *   **Stores this `Request` in `pendingRequests`**.
        *   Constructs the `EVALUATE_SCRIPT` WebSocket message map: `{ "type": "EVALUATE_SCRIPT", "requestId": req.Id, "scriptToEvaluate": script }`.
        *   Uses `BroadcastToAgent` to send it.
        *   Returns the `Request` object (its `recvChan` is used by the HTTP handler if waiting).
    *   **`BroadcastToAgent(clientId string, payload map[string]any)`**: Sends the payload to the fanout for the specified `clientId`.

5.  **`NewServeMux()`**:
    *   Configures the HTTP routing for the agent server.
    *   `GET /agents/{clientId}/subscribe`: Handles WebSocket upgrade requests using `gohttp.WSServe(handler, nil)`.
    *   `POST /agents/{clientId}/eval`:
        *   This is the primary endpoint for external systems (like the `vibrant client` CLI) to trigger script evaluations.
        *   Reads the **raw request body as the JavaScript string** to be evaluated.
        *   Calls `handler.SubmitEvalRequest()` to send the script to the client via WebSocket.
        *   If the query parameter `?wait=true` is present, this HTTP handler blocks and waits for a message on the `Request.recvChan` (with a timeout). It then returns the script's result (or error) in its HTTP response.
        *   If `?wait=true` is not present, it responds immediately with the `requestId`.
    *   `GET /test_eval?agent=<name>&script=<javascript>`: A simple GET endpoint for testing the `EVALUATE_SCRIPT` flow without needing a POST body.

### Workflow Summary

1.  The Go agent server (started by `vibrant agents serve`) listens (default `localhost:9999`).
2.  Chrome DevTools extension connects to `GET /agents/{clientId}/subscribe`. `Conn.OnStart` is called, a welcome script is sent.
3.  An external trigger (e.g., `vibrant client ...` CLI, or another service) makes a `POST /agents/{clientId}/eval` request with raw JavaScript in the body.
4.  The HTTP handler calls `handler.SubmitEvalRequest`. This generates a `requestId`, stores the request, and broadcasts an `EVALUATE_SCRIPT` message over WebSocket to the specified `clientId`.
5.  The Chrome extension receives the `EVALUATE_SCRIPT` message, executes the script on the page, and sends an `EVALUATION_RESULT` message (with the same `requestId` and the script's output/error) back over WebSocket.
6.  `Conn.HandleMessage` on the server receives this result, finds the pending `Request` by `requestId`, and sends the result to the `recvChan` of that `Request`.
7.  If the original `POST /agents/{clientId}/eval` request included `?wait=true`, its handler receives the result from `recvChan` and includes it in the HTTP response.
