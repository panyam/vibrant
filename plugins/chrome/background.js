let activeWebSockets = {}; // { tabId: { ws: WebSocket, panelPort: Port, connectionName: "..." } }

chrome.runtime.onConnect.addListener(function(port) {
  if (port.name && port.name.startsWith("devtools-panel-")) {
    const tabId = parseInt(port.name.split("-").pop(), 10);
    console.log(`Background: DevTools panel connected for tab ${tabId}`);

    // Associate port with its tabId early, even before WebSocket connection
    if (activeWebSockets[tabId]) {
        activeWebSockets[tabId].panelPort = port;
    } else {
        activeWebSockets[tabId] = { ws: null, panelPort: port, connectionName: null };
    }

    port.onMessage.addListener(function(message) {
      console.log(`Background: Received message from panel for tab ${tabId}:`, message);
      const currentConnection = activeWebSockets[tabId];

      if (message.type === "CONNECT_WEBSOCKET") {
        if (currentConnection && currentConnection.ws && currentConnection.ws.readyState === WebSocket.OPEN) {
          if (currentConnection.connectionName === message.connectionName) {
            console.log(`Background: WebSocket already open for ${message.connectionName} on tab ${tabId}`);
            if (currentConnection.panelPort) {
              currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: message.connectionName });
            }
            return;
          } else {
            console.log(`Background: Closing existing WebSocket for ${currentConnection.connectionName} on tab ${tabId} to open new one for ${message.connectionName}.`);
            currentConnection.ws.close();
          }
        }

        const connectionName = message.connectionName;
        const wsUrl = `ws://localhost:9999/agents/${connectionName}/subscribe`; // Ensure this port is correct
        console.log(`Background: Attempting to connect WebSocket to ${wsUrl} for tab ${tabId}`);

        try {
          const ws = new WebSocket(wsUrl);
          activeWebSockets[tabId] = { ...currentConnection, ws: ws, connectionName: connectionName, panelPort: port };

          ws.onopen = function() {
            console.log(`Background: WebSocket connected for ${connectionName} on tab ${tabId}`);
            const connInfo = activeWebSockets[tabId]; // Get latest info
            if (connInfo && connInfo.panelPort) {
              connInfo.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: connectionName });
            }
          };

          ws.onmessage = function(event) {
            console.log(`Background: WebSocket message received for ${connectionName} on tab ${tabId}:`, event.data); // Can be verbose
            const connInfo = activeWebSockets[tabId];
            if (connInfo && connInfo.panelPort) {
              try {
                const jsonData = JSON.parse(event.data);
                connInfo.panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: jsonData });
              } catch (e) {
                connInfo.panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: event.data });
                console.warn("Background: WebSocket message was not valid JSON, sent as string.", e, event.data);
              }
            }
          };

          ws.onerror = function(error) {
            console.error(`Background: WebSocket error for ${connectionName} on tab ${tabId}:`, error);
            const connInfo = activeWebSockets[tabId];
            if (connInfo && connInfo.panelPort) {
              connInfo.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: `Error: ${error.message || 'Connection failed'}`, connectionName: connectionName });
            }
            if (connInfo) connInfo.ws = null; 
          };

          ws.onclose = function(event) {
            console.log(`Background: WebSocket closed for ${connectionName} on tab ${tabId}. Code: ${event.code}, Reason: ${event.reason}`);
            const connInfo = activeWebSockets[tabId];
            if (connInfo && connInfo.panelPort) {
              connInfo.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Disconnected", connectionName: connectionName, wasClean: event.wasClean });
            }
            // Important: Only nullify ws and connectionName, don't delete activeWebSockets[tabId] entirely
            // as panelPort might still be needed or a new ws might be established.
            if (connInfo) {
                connInfo.ws = null;
                // Keep connectionName if you want to show it in panel even when disconnected,
                // or nullify it if disconnect means "forget this connection attempt"
                // connInfo.connectionName = null; 
            }
          };
        } catch (e) {
            console.error(`Background: Failed to create WebSocket object for ${connectionName} on tab ${tabId}:`, e);
            if (currentConnection && currentConnection.panelPort) {
                currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: `Error: Failed to create WebSocket`, connectionName: connectionName });
            }
        }

      } else if (message.type === "DISCONNECT_WEBSOCKET") {
        if (currentConnection && currentConnection.ws && currentConnection.connectionName === message.connectionName) {
          console.log(`Background: Closing WebSocket for ${message.connectionName} on tab ${tabId} due to panel request.`);
          currentConnection.ws.close();
          // Status update will be sent by ws.onclose handler
        } else {
          console.warn(`Background: No matching WebSocket to disconnect for ${message.connectionName} on tab ${tabId}`);
          if(currentConnection && currentConnection.panelPort) {
            currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Not Connected or different connection", connectionName: message.connectionName });
          }
        }
      } else if (message.type === "FORWARD_TO_WEBSOCKET_SERVER") { // New: Handle forwarding from panel to WS
        const connInfo = activeWebSockets[tabId];
        if (connInfo && connInfo.ws && connInfo.ws.readyState === WebSocket.OPEN) {
            try {
                const messageString = JSON.stringify(message.payload);
                connInfo.ws.send(messageString);
                console.log(`Background: Forwarded message to WebSocket server for tab ${tabId}:`, message.payload);
            } catch (e) {
                console.error(`Background: Error sending message to WebSocket for tab ${tabId}:`, e, message.payload);
            }
        } else {
            console.warn(`Background: WebSocket not available/open for tab ${tabId}. Cannot forward message:`, message.payload);
        }
      }
    });

    port.onDisconnect.addListener(function() {
      console.log(`Background: DevTools panel disconnected for tab ${tabId}.`);
      const connInfo = activeWebSockets[tabId];
      if (connInfo) {
        connInfo.panelPort = null; // Clear the port reference.
        // Decide if WebSocket should be closed when panel disconnects.
        // If we want to keep it alive for potential panel reopen:
        // console.log(`Background: Panel for tab ${tabId} disconnected, but WebSocket (if any) remains.`);
        // If we want to close it:
        /*
        if (connInfo.ws) {
            console.log(`Background: Closing WebSocket for tab ${tabId} because panel disconnected.`);
            connInfo.ws.close();
            connInfo.ws = null;
            connInfo.connectionName = null; 
            // Consider if activeWebSockets[tabId] should be deleted or kept with ws = null
        }
        */
      }
    });
  }
});

chrome.tabs.onRemoved.addListener(function(tabId, removeInfo) {
  if (activeWebSockets[tabId]) {
    console.log(`Background: Tab ${tabId} removed. Closing WebSocket if active.`);
    if (activeWebSockets[tabId].ws) {
      activeWebSockets[tabId].ws.close();
    }
    delete activeWebSockets[tabId]; // Clean up the entry for the closed tab
  }
});

chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
  // If the tab navigates to a new URL (and not just a subframe or something)
  if (changeInfo.status === 'loading' && changeInfo.url) { // More reliable check for navigation
    if (activeWebSockets[tabId] && activeWebSockets[tabId].ws) {
      console.log(`Background: Tab ${tabId} navigated to ${changeInfo.url}. Closing WebSocket.`);
      activeWebSockets[tabId].ws.close(); // This will trigger onclose which updates panel status
      // No need to delete activeWebSockets[tabId] here, let onclose and panel handle state
    }
  }
});

console.log("Background service worker started.");
