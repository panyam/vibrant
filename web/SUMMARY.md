## Go Web Service Summary (`./web`)

This directory contains the Go package responsible for handling backend HTTP and WebSocket communications, primarily for the agent interaction feature.

### Purpose

*   **WebSocket Server**: Provides a WebSocket endpoint for the Chrome DevTools extension to connect to.
*   **Command Ingestion**: Offers HTTP POST endpoints to receive commands that are then broadcasted to connected WebSocket clients (i.e., the Chrome extension instances).
*   **Client Management**: Manages active WebSocket connections.

### Key Components & Functionality

**`server.go`**:

1.  **Package `web`**: Defines the reusable web service components.

2.  **WebSocket Upgrader (`upgrader`)**:
    *   Uses `github.com/gorilla/websocket`.
    *   Configured with a basic `CheckOrigin` function that allows all origins (suitable for local development).

3.  **Connection Manager (`connectionManager`, `manager`)**:
    *   A struct (`connectionManager`) to manage active WebSocket connections, using a `sync.RWMutex` for thread safety.
    *   Stores connections in a `map[string]map[*websocket.Conn]bool`, allowing multiple DevTools panels (clients) to connect using the same `connectionName` (though typically, a `connectionName` might be unique per agent or debugging session).
    *   `AddConnection(connectionName, conn)`: Registers a new client.
    *   `RemoveConnection(connectionName, conn)`: Unregisters a client upon disconnection.
    *   `BroadcastToAgent(connectionName, messageType, message)`: Sends a given message (byte slice) to all clients subscribed under a specific `connectionName`.
    *   A global instance `manager` of `connectionManager` is used.
    *   `GetConnectionManager()`: An exported function to allow other Go packages to access the global `manager` if direct broadcasting is needed (though using dedicated API endpoints is generally preferred).

4.  **WebSocket Handler (`serveWs`)**:
    *   Uses the websocket (ws) module in github.com/panyam/goutils/http package.
    *   Takes care of upgrades etc and provides easy abstractions to deal with Websocket messages and broadcasts.
    *   Once a connection is created it is added to the list of Fanout objects (responsible for fanning out messages to
        all listeners).
    *   Sends a welcome message to the newly connected client.

5.  **HTTP Command Handler (`handleAgentCommandPost`)**:
    *   Path: Expected to handle POST requests like `/agents/{connectionName}/{COMMAND_TYPE}` (relative to the mount point).
    *   Validates the request method (POST only).
    *   Parses `connectionName` and `COMMAND_TYPE` from the URL path.
    *   Reads the JSON body of the POST request.
    *   Constructs the final WebSocket message payload:
        *   Uses the parsed JSON body as the base.
        *   Sets the `"type"` field of the payload to the `COMMAND_TYPE` extracted from the URL.
        *   If `COMMAND_TYPE` is `QUERY_SELECTOR_ALL` and no `requestId` is in the body, a unique `requestId` (timestamp-based) is added.
    *   Uses `manager.BroadcastToAgent()` to send this final JSON message to the relevant connected client(s).
    *   Responds to the HTTP POST request with a JSON status message.

6.  **ServeMux Creation (`NewServeMux()`)**:
    *   This is the primary exported function to set up the routing for this package.
    *   Creates an `http.ServeMux`.
    *   Registers handlers:
        *   `/agents/`: Routes to `serveWs` for WebSocket connections.
        *   `/agents/`: Routes to `handleAgentCommandPost` for incoming HTTP POST commands.
        *   `/test_broadcast`: A GET endpoint for simple testing of broadcasting messages. It accepts `agent`, `msg`, and `type` query parameters.
    *   This `ServeMux` is intended to be used by a higher-level command (e.g., `vibrant agents serve`) to start an HTTP server.

### Workflow Summary

1.  An HTTP server (started by `vibrant agents serve` using `web.NewServeMux()`) listens, typically on `localhost:9999`.
2.  The Chrome DevTools extension initiates a WebSocket connection to `ws://localhost:9999/agents/<connectionName>/subscribe`.
3.  `serveWs` handles this, upgrades the connection, and registers it with the `manager`.
4.  External processes (or the test endpoint) can make HTTP POST requests to `http://localhost:9999/agents/<connectionName>/<COMMAND_TYPE>` with a JSON body.
5.  `handleAgentCommandPost` receives this, forms a complete WebSocket message (ensuring `type` matches `COMMAND_TYPE`), and uses `manager.BroadcastToAgent`.
6.  The message is sent to all Chrome extension instances connected with the specified `connectionName`.
