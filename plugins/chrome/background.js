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
            currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: message.connectionName });
            return;
          } else {
            console.log(`Background: Closing existing WebSocket for ${currentConnection.connectionName} on tab ${tabId} to open new one.`);
            currentConnection.ws.close(); // Close existing if name changes
          }
        }

        const connectionName = message.connectionName;
        const wsUrl = `ws://localhost:9999/agent/${connectionName}/subscribe`;
        console.log(`Background: Attempting to connect WebSocket to ${wsUrl} for tab ${tabId}`);

        try {
          const ws = new WebSocket(wsUrl);
          activeWebSockets[tabId] = { ...activeWebSockets[tabId], ws: ws, connectionName: connectionName };

          ws.onopen = function() {
            console.log(`Background: WebSocket connected for ${connectionName} on tab ${tabId}`);
            if (activeWebSockets[tabId] && activeWebSockets[tabId].panelPort) {
              activeWebSockets[tabId].panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: connectionName });
            }
            // Store connection name for potential later use (e.g., re-establishing after DevTools closes/opens)
            // chrome.storage.local.set({[`lastConnectionName_${tabId}`]: connectionName});
          };

          ws.onmessage = function(event) {
            console.log(`Background: WebSocket message received for ${connectionName} on tab ${tabId}:`, event.data);
            if (activeWebSockets[tabId] && activeWebSockets[tabId].panelPort) {
              try {
                const jsonData = JSON.parse(event.data);
                activeWebSockets[tabId].panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: jsonData });
              } catch (e) {
                // If data is not JSON, send as is or handle error
                activeWebSockets[tabId].panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: event.data });
                console.warn("Background: WebSocket message was not valid JSON, sent as string.", e);
              }
            }
          };

          ws.onerror = function(error) {
            console.error(`Background: WebSocket error for ${connectionName} on tab ${tabId}:`, error);
            if (activeWebSockets[tabId] && activeWebSockets[tabId].panelPort) {
              activeWebSockets[tabId].panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: `Error: ${error.message || 'Connection failed'}`, connectionName: connectionName });
            }
            if (activeWebSockets[tabId]) activeWebSockets[tabId].ws = null; // Clear the ws object
          };

          ws.onclose = function(event) {
            console.log(`Background: WebSocket closed for ${connectionName} on tab ${tabId}. Code: ${event.code}, Reason: ${event.reason}`);
            if (activeWebSockets[tabId] && activeWebSockets[tabId].panelPort) {
              activeWebSockets[tabId].panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Disconnected", connectionName: connectionName, wasClean: event.wasClean });
            }
            if (activeWebSockets[tabId]) {
                activeWebSockets[tabId].ws = null; // Clear the ws object
                activeWebSockets[tabId].connectionName = null;
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
      }
    });

    port.onDisconnect.addListener(function() {
      console.log(`Background: DevTools panel disconnected for tab ${tabId}.`);
      // If the panel disconnects, we might want to close the WebSocket if it's specific to that panel instance.
      // However, the user might just be closing and reopening the panel.
      // For now, we keep the WebSocket alive if it exists, and the panel can reconnect.
      // To auto-close: uncomment below
      /*
      const conn = activeWebSockets[tabId];
      if (conn && conn.ws) {
        console.log(`Background: Closing WebSocket for tab ${tabId} because panel disconnected.`);
        conn.ws.close();
        // delete activeWebSockets[tabId]; // Or just nullify ws and connectionName
      }
      */
     if(activeWebSockets[tabId]){
        activeWebSockets[tabId].panelPort = null; // Crucial: clear the port reference
     }
    });
  }
});

// Clean up WebSockets for closed tabs
chrome.tabs.onRemoved.addListener(function(tabId, removeInfo) {
  if (activeWebSockets[tabId]) {
    console.log(`Background: Tab ${tabId} removed. Closing WebSocket if active.`);
    if (activeWebSockets[tabId].ws) {
      activeWebSockets[tabId].ws.close();
    }
    delete activeWebSockets[tabId];
  }
});

// Optional: Handle navigations within a tab (might want to disconnect/reset)
chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
  // If the tab navigates to a new URL (and not just a subframe or something)
  if (changeInfo.url && activeWebSockets[tabId] && activeWebSockets[tabId].ws) {
    console.log(`Background: Tab ${tabId} updated (navigated to ${changeInfo.url}). Closing WebSocket.`);
    activeWebSockets[tabId].ws.close();
    // The panel will show disconnected and user can reconnect on the new page.
  }
});

console.log("Background service worker started.");
