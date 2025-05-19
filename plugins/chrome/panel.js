const connectionNameInput =  document.getElementById('connectionName');
const connectButton =  document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null; 

// Reconnection logic variables
let reconnectAttempts = 0;
let reconnectTimeoutId = null;
let userInitiatedDisconnect = false; 
const INITIAL_RECONNECT_DELAY = 1000; 
const MAX_RECONNECT_DELAY = 10000;    

// Minimal logging to panel's own console for critical errors or status.
// Page interactions are now fully driven by scripts from the backend.

function handleEvaluateScriptRequest(requestDetails) {
    if (!requestDetails || !requestDetails.requestId || typeof requestDetails.scriptToEvaluate !== 'string') {
        console.error("Panel: Invalid EVALUATE_SCRIPT request", requestDetails);
        // Optionally, send an error back to the server if panelPort exists
        if (panelPort) {
            panelPort.postMessage({
                type: "FORWARD_TO_WEBSOCKET_SERVER",
                payload: {
                    type: "EVALUATION_RESULT",
                    requestId: requestDetails.requestId || "unknown",
                    error: "Invalid EVALUATE_SCRIPT request structure from backend",
                    isException: true,
                    exceptionInfo: "Invalid request structure."
                }
            });
        }
        return;
    }
    // console.log("Panel: Received EVALUATE_SCRIPT request", requestDetails.requestId, "Script:", requestDetails.scriptToEvaluate); // Can be very verbose

    chrome.devtools.inspectedWindow.eval(
        requestDetails.scriptToEvaluate,
        function(result, isExceptionInfo) { // Note: isExceptionInfo is an object if an error occurred
            // console.log(`Panel: Evaluation complete for requestId: ${requestDetails.requestId}. Exception:`, isExceptionInfo, "Result:", result); // Verbose
            
            let exceptionDetails = null;
            if (isExceptionInfo) {
                // isExceptionInfo can be an object with description, or just a value if script threw non-Error
                if (typeof isExceptionInfo === 'object' && isExceptionInfo !== null) {
                    exceptionDetails = isExceptionInfo.description || isExceptionInfo.value || JSON.stringify(isExceptionInfo);
                } else {
                    exceptionDetails = String(isExceptionInfo);
                }
                 console.error(`Panel: Exception during script evaluation (ReqID: ${requestDetails.requestId}):`, exceptionDetails, "Original script:", requestDetails.scriptToEvaluate);
            }

            const responsePayload = {
                type: "EVALUATION_RESULT",
                requestId: requestDetails.requestId,
                result: result, // This will be undefined if an exception occurred and not caught by user script
                isException: !!isExceptionInfo, // Coerce to boolean
                exceptionInfo: exceptionDetails
            };

            if (panelPort) {
                panelPort.postMessage({
                    type: "FORWARD_TO_WEBSOCKET_SERVER",
                    payload: responsePayload
                });
                // console.log("Panel: Sent EVALUATION_RESULT to background for forwarding:", responsePayload); // Verbose
            } else {
                console.error("Panel: panelPort is null, cannot send EVALUATION_RESULT.");
            }
        }
    );
}

function scheduleReconnect() {
    if (userInitiatedDisconnect) return;
    if (!currentConnectionName) {
        if (statusDiv) statusDiv.textContent = "Status: Disconnected. Enter name and connect.";
        if (connectButton) connectButton.style.display = 'inline-block';
        if (disconnectButton) disconnectButton.style.display = 'none';
        if (connectionNameInput) connectionNameInput.disabled = false;
        return;
    }
    reconnectAttempts++;
    let delay = INITIAL_RECONNECT_DELAY * Math.pow(2, reconnectAttempts -1 );
    delay = Math.min(delay, MAX_RECONNECT_DELAY);
    if (statusDiv) statusDiv.textContent = `Status: Disconnected. Reconnecting in ${Math.round(delay/1000)}s... (Attempt ${reconnectAttempts})`;
    clearTimeout(reconnectTimeoutId);
    reconnectTimeoutId = setTimeout(() => connect(false), delay);
}

function clearReconnectTimer() {
    clearTimeout(reconnectTimeoutId);
    reconnectTimeoutId = null;
}

function connect(isUserClick = true) {
  clearReconnectTimer();
  if (isUserClick) {
    userInitiatedDisconnect = false;
    reconnectAttempts = 0;
  }

  if (!panelPort) {
    try {
        panelPort = chrome.runtime.connect({ name: "devtools-panel-" + chrome.devtools.inspectedWindow.tabId });
        console.log("Panel: Port (re)established with background script.");

        panelPort.onMessage.addListener(function(message) {
          // console.log("Panel: Received message from background:", message); // Verbose
          if (message.type === "WEBSOCKET_MESSAGE") {
            if (message.data) {
                if (message.data.type === 'EVALUATE_SCRIPT') { // The primary message type now
                    handleEvaluateScriptRequest(message.data);
                } else {
                    // Could log unhandled message types from server if necessary
                    console.warn("Panel: Received unhandled WebSocket message data type:", message.data.type, message.data);
                }
            } else {
                console.warn("Panel: Received empty data in WEBSOCKET_MESSAGE from background.");
            }
          } else if (message.type === "WEBSOCKET_STATUS") {
            if (statusDiv) statusDiv.textContent = `Status: ${message.status}`; 
            if (message.status === "Connected") {
              if (connectButton) connectButton.style.display = 'none';
              if (disconnectButton) disconnectButton.style.display = 'inline-block';
              if (connectionNameInput) connectionNameInput.disabled = true;
              userInitiatedDisconnect = false;
              reconnectAttempts = 0;
              clearReconnectTimer();
            } else { 
              if (connectButton) connectButton.style.display = 'inline-block';
              if (disconnectButton) disconnectButton.style.display = 'none';
              if (connectionNameInput) connectionNameInput.disabled = false;
              if (!userInitiatedDisconnect && (message.status === "Disconnected" || message.status.startsWith("Error:"))) {
                 scheduleReconnect();
              } else if (userInitiatedDisconnect) {
                 currentConnectionName = null; 
              }
            }
          }
        });

        panelPort.onDisconnect.addListener(function() {
          console.warn("Panel: Port to background script disconnected.");
          if (statusDiv) statusDiv.textContent = "Status: Disconnected from background script.";
          panelPort = null; 
          if (connectButton) connectButton.style.display = 'inline-block';
          if (disconnectButton) disconnectButton.style.display = 'none';
          if (connectionNameInput) connectionNameInput.disabled = false;
          if (!userInitiatedDisconnect && currentConnectionName) {
              scheduleReconnect();
          }
        });
    } catch (e) {
        console.error("Panel: Failed to connect port to background script:", e);
        if (statusDiv) statusDiv.textContent = "Status: Error connecting to background.";
        return; 
    }
  }

  const nameToConnect = connectionNameInput ? connectionNameInput.value.trim() : "";
  if (nameToConnect) {
    currentConnectionName = nameToConnect; 
    console.log(`Panel: Attempting to connect WebSocket for name: ${currentConnectionName}`);
    if (panelPort) {
        panelPort.postMessage({
          type: "CONNECT_WEBSOCKET",
          tabId: chrome.devtools.inspectedWindow.tabId,
          connectionName: currentConnectionName
        });
        if (statusDiv) {
             statusDiv.textContent = (isUserClick || reconnectAttempts === 0) ? "Status: Connecting..." : `Status: Reconnecting... (Attempt ${reconnectAttempts})`;
        }
    } else {
        console.error("Panel: panelPort is null, cannot send CONNECT_WEBSOCKET.");
        if(statusDiv) statusDiv.textContent = "Status: Error - background connection lost.";
    }
  } else {
    if (isUserClick && statusDiv) statusDiv.textContent = "Status: Please enter a connection name.";
  }
}

function disconnect() {
  userInitiatedDisconnect = true; 
  clearReconnectTimer(); 
  reconnectAttempts = 0;
  if (panelPort && currentConnectionName) {
    console.log(`Panel: User initiated disconnect for ${currentConnectionName}`);
    panelPort.postMessage({
      type: "DISCONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId,
      connectionName: currentConnectionName 
    });
  } else {
    if (statusDiv) statusDiv.textContent = "Status: Not Connected";
    if (connectButton) connectButton.style.display = 'inline-block';
    if (disconnectButton) disconnectButton.style.display = 'none';
    if (connectionNameInput) connectionNameInput.disabled = false;
  }
}

if (connectButton) connectButton.addEventListener('click', () => connect(true));
if (disconnectButton) disconnectButton.addEventListener('click', disconnect);
console.log("Panel.js loaded for tab: " + chrome.devtools.inspectedWindow.tabId);

const storageKey = `lastConnectionName_${chrome.devtools.inspectedWindow.tabId}`;
const savedName = sessionStorage.getItem(storageKey);
if (savedName && connectionNameInput) {
    connectionNameInput.value = savedName;
}
if (connectionNameInput) {
    connectionNameInput.addEventListener('input', (event) => { 
        if (event.target.value) {
            sessionStorage.setItem(storageKey, event.target.value);
        } else {
            sessionStorage.removeItem(storageKey);
        }
    });
}
