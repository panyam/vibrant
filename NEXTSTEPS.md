# Next Steps & Outstanding Work for `vibrant`

This document outlines potential areas for future development, improvements, and features based on the current "remote evaluation" architecture.

## I. Core Remote Evaluation & Extension Enhancements

1.  **Complex Result Handling from `EVALUATE_SCRIPT`**:
    *   **Problem**: `chrome.devtools.inspectedWindow.eval` has limitations on the complexity and serializability of the `result` it can directly return. Large or complex objects (like DOM elements themselves) might not transfer well.
    *   **Next Step**: For scripts that need to return complex data, the script itself should serialize its result into a JSON string (e.g., `return JSON.stringify(complexObject);`). The Go backend (`Conn.HandleMessage`) would then need to `json.Unmarshal` this string if it expects a complex type. This is partially handled for simple objects, but needs to be robust for diverse return types.

2.  **Security of Evaluated Scripts**:
    *   **Problem**: The backend has full control to evaluate any JavaScript. While powerful, this requires trusting the source of the scripts (i.e., the Go backend logic).
    *   **Next Step**: This is inherent to the design. Maintain good practices on the backend for constructing scripts and handling inputs that might become part of those scripts. No direct changes to the extension are needed for this, but it's an architectural consideration.

3.  **Streaming Large Results or Continuous Data**:
    *   **Problem**: The current `EVALUATION_RESULT` is for a single script execution. If a script needs to send multiple updates or stream data, this model isn't ideal.
    *   **Next Step**: For such scenarios, the evaluated script could use `console.log` with specific prefixes. A separate, lightweight "log scraper" script (or a more complex `EVALUATE_SCRIPT` that sets up listeners and then calls back repeatedly via new `EVALUATE_SCRIPT` calls for a "send message" action) could be considered if true streaming from the page context to the Go backend is needed without a new WebSocket. (This deviates from the simple eval model). *Alternatively, the Go backend can just issue many small `EVALUATE_SCRIPT` calls.*

4.  **Error Handling in `panel.js` for Eval**:
    *   The `exceptionInfo` in `EVALUATION_RESULT` is good. Ensure it's comprehensive.
    *   Consider if the Go backend needs a way to know if the `chrome.devtools.inspectedWindow.eval` call *itself* failed before the script even ran (less likely but possible).

## II. Go Application (Backend & CLI)

1.  **Go Agent Logic for `EVALUATION_RESULT` (`web/server.go`)**:
    *   **TODO**: The `Conn.HandleMessage` function currently just logs `EVALUATION_RESULT`. This needs to be connected to the actual Go logic that initiated the `EVALUATE_SCRIPT` request.
    *   **Next Step**: Implement a robust mechanism (e.g., using the `requestId`, a map of waiting goroutines/channels, or a pub/sub system within the Go agent) to deliver results back to the specific part of the Go code that is interested in them, especially for the `wait=true` scenario on the `/eval` endpoint. The current `req.recvChan` handles this for direct HTTP waits, but more general agent logic might need a different pattern.

2.  **Request Timeouts & Cleanup (`web/server.go`)**:
    *   **Problem**: If an `EVALUATE_SCRIPT` is sent, but the extension never sends back an `EVALUATION_RESULT` (e.g., panel closed, browser crashed), the `Request` object might remain in `pendingRequests` indefinitely.
    *   **Next Step**: Implement a timeout mechanism for requests in `pendingRequests`. When a request times out, its `recvChan` should be closed (if `wait=true` was used) and the request removed from the map to prevent memory leaks. The `wait=true` HTTP handler has a timeout, but `pendingRequests` for non-waiting calls also need cleanup.

3.  **State Management in Go Agent**:
    *   The Go agent will likely need to maintain state based on the results of script evaluations to make intelligent decisions about subsequent scripts to send. Design how this state is managed.

4.  **CLI (`cmd/client.go`)**:
    *   **More Commands**: Add more client subcommands that leverage the `sendEvalScript` function to perform common or useful page interactions.
    *   **Output Formatting**: For commands that use `wait=true` and get results, improve the CLI output formatting for different types of results (JSON, strings, numbers).
    *   **Error Handling**: Improve user-facing error messages in the CLI when `sendEvalScript` fails.

5.  **Security for Agent Server**:
    *   If the agent server (`:9999`) is exposed beyond `localhost`, implement authentication and authorization for both WebSocket connections and the `/agents/.../eval` HTTP endpoint.

## III. Chrome Extension UI/UX (Panel)

1.  **Displaying Activity**:
    *   Consider adding a small, unobtrusive log within the panel itself to show recent `EVALUATE_SCRIPT` requests sent and `EVALUATION_RESULT`s received (or at least their `requestId`s and status). This would give the user more visibility than just relying on console logs.
2.  **Manual Script Input (Advanced)**:
    *   Optionally, add a textarea in the panel where a developer could manually type JavaScript to be evaluated, using the same `handleEvaluateScriptRequest` pathway (it would generate a local `requestId`).

## IV. Testing

1.  **Go Unit/Integration Tests**: Write tests for the `web` package, especially the request/response lifecycle for `/eval` and WebSocket message handling.
2.  **CLI Tests**: Test the `cmd/client.go` commands.
3.  **Extension Testing**: Manual testing is primary now. Automated testing (e.g., Puppeteer controlling a browser with the extension loaded) is a larger effort but valuable long-term.

## Suggested Starting Points for Next Steps

1.  **Robust `EVALUATION_RESULT` Handling in `web/server.go`**: Ensure `Conn.HandleMessage` correctly and reliably dispatches results to waiting goroutines (especially for the `wait=true` HTTP calls via `recvChan`) and cleans up `pendingRequests`. This is critical for the backend to be useful.
2.  **Implement Request Timeouts/Cleanup in `pendingRequests`**: Prevent memory leaks on the Go server for requests that never get a response.
3.  **Add More `vibrant client` Commands**: Build out the CLI with useful canned JavaScript evaluations to showcase the power of the remote eval system (e.g., get specific element attributes, check if an element exists, count elements).
4.  **Panel Activity Log**: A simple log in the panel UI would improve debuggability.
