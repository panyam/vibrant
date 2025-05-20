
let activeWebSockets = {}; // { tabId: { ws: WebSocket, panelPort: Port, connectionName: "..." } }

chrome.runtime.onConnect.addListener(function(port) {
  if (port.name && port.name.startsWith("devtools-panel-")) {
    const tabIdFromPortName = parseInt(port.name.split("-").pop(), 10);
    console.log(`Background: DevTools panel connected for tab ${tabIdFromPortName}`);

    if (!activeWebSockets[tabIdFromPortName]) {
        activeWebSockets[tabIdFromPortName] = { ws: null, panelPort: null, connectionName: null };
    }
    activeWebSockets[tabIdFromPortName].panelPort = port;

    port.onMessage.addListener(function(message) {
      const currentConnection = activeWebSockets[tabIdFromPortName];
      if (!currentConnection) {
          console.warn(`Background: Received message for disconnected or unknown panel (tabId: ${tabIdFromPortName})`, message);
          return;
      }
      console.log(`Background: Received message from panel for tab ${tabIdFromPortName}:`, message);

      if (message.type === "CONNECT_WEBSOCKET") {
        // ... (existing WebSocket connection logic remains the same) ...
        if (currentConnection.ws && currentConnection.ws.readyState === WebSocket.OPEN) {
          if (currentConnection.connectionName === message.connectionName) {
            console.log(`Background: WebSocket already open for ${message.connectionName} on tab ${tabIdFromPortName}`);
            if (currentConnection.panelPort) {
              currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: message.connectionName });
            }
            return;
          } else {
            console.log(`Background: Closing existing WebSocket for ${currentConnection.connectionName} on tab ${tabIdFromPortName} to open new one for ${message.connectionName}.`);
            currentConnection.ws.close();
          }
        }
        const connectionName = message.connectionName;
        const wsUrl = `ws://localhost:9999/agents/${connectionName}/subscribe`;
        console.log(`Background: Attempting to connect WebSocket to ${wsUrl} for tab ${tabIdFromPortName}`);
        try {
          const ws = new WebSocket(wsUrl);
          currentConnection.ws = ws;
          currentConnection.connectionName = connectionName;
          ws.onopen = function() {
            console.log(`Background: WebSocket connected for ${connectionName} on tab ${tabIdFromPortName}`);
            if (currentConnection.panelPort) {
              currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Connected", connectionName: connectionName });
            }
          };
          ws.onmessage = function(event) {
            if (currentConnection.panelPort) {
              try {
                const jsonData = JSON.parse(event.data);
                currentConnection.panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: jsonData });
              } catch (e) {
                currentConnection.panelPort.postMessage({ type: "WEBSOCKET_MESSAGE", data: event.data });
                console.warn("Background: WebSocket message was not valid JSON, sent as string.", e, event.data);
              }
            }
          };
          ws.onerror = function(error) {
            console.error(`Background: WebSocket error for ${connectionName} on tab ${tabIdFromPortName}:`, error);
            if (currentConnection.panelPort) {
              currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: `Error: ${error.message || 'Connection failed'}`, connectionName: connectionName });
            }
            currentConnection.ws = null;
          };
          ws.onclose = function(event) {
            console.log(`Background: WebSocket closed for ${connectionName} on tab ${tabIdFromPortName}. Code: ${event.code}, Reason: ${event.reason}`);
            if (currentConnection.panelPort) {
              currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Disconnected", connectionName: connectionName, wasClean: event.wasClean });
            }
            currentConnection.ws = null;
          };
        } catch (e) {
            console.error(`Background: Failed to create WebSocket object for ${connectionName} on tab ${tabIdFromPortName}:`, e);
            if (currentConnection.panelPort) {
                currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: `Error: Failed to create WebSocket`, connectionName: connectionName });
            }
        }
      } else if (message.type === "DISCONNECT_WEBSOCKET") {
        // ... (existing WebSocket disconnect logic remains the same) ...
        if (currentConnection.ws && currentConnection.connectionName === message.connectionName) {
          console.log(`Background: Closing WebSocket for ${message.connectionName} on tab ${tabIdFromPortName} due to panel request.`);
          currentConnection.ws.close();
        } else {
          console.warn(`Background: No matching WebSocket to disconnect for ${message.connectionName} on tab ${tabIdFromPortName}`);
          if(currentConnection.panelPort) {
            currentConnection.panelPort.postMessage({ type: "WEBSOCKET_STATUS", status: "Not Connected or different connection", connectionName: message.connectionName });
          }
        }
      } else if (message.type === "FORWARD_TO_WEBSOCKET_SERVER") {
        // ... (existing forwarding logic remains the same) ...
        if (currentConnection.ws && currentConnection.ws.readyState === WebSocket.OPEN) {
            try {
                const messageString = JSON.stringify(message.payload);
                currentConnection.ws.send(messageString);
            } catch (e) {
                console.error(`Background: Error sending message to WebSocket for tab ${tabIdFromPortName}:`, e, message.payload);
            }
        } else {
            console.warn(`Background: WebSocket not available/open for tab ${tabIdFromPortName}. Cannot forward message:`, message.payload);
        }
      } else if (message.type === "REQUEST_TAB_CAPTURE") {
        console.log(`Background: Received REQUEST_TAB_CAPTURE for inspectedTabId ${message.tabId}, requestId: ${message.requestId}`);
        
        // Get the windowId of the inspected tab
        chrome.tabs.get(message.tabId, (inspectedTabDetails) => {
            if (chrome.runtime.lastError) {
                console.error(`Background: Error getting details for inspectedTabId ${message.tabId}:`, chrome.runtime.lastError.message);
                if (currentConnection.panelPort) {
                    currentConnection.panelPort.postMessage({
                        type: "TAB_CAPTURE_COMPLETE",
                        requestId: message.requestId,
                        dataUrl: null,
                        error: `Failed to get details for inspected tab: ${chrome.runtime.lastError.message}`
                    });
                }
                return;
            }

            if (!inspectedTabDetails) {
                 console.error(`Background: No details returned for inspectedTabId ${message.tabId}.`);
                 if (currentConnection.panelPort) {
                    currentConnection.panelPort.postMessage({
                        type: "TAB_CAPTURE_COMPLETE",
                        requestId: message.requestId,
                        dataUrl: null,
                        error: `No details returned for inspected tab ${message.tabId}.`
                    });
                }
                return;
            }

            const windowToCapture = inspectedTabDetails.windowId;
            console.log(`Background: Attempting to capture active tab in windowId ${windowToCapture} (inspectedTabId ${message.tabId}) for requestId: ${message.requestId}`);

            chrome.tabs.captureVisibleTab(windowToCapture, { format: "png" }, (dataUrl) => {
                let response = {
                    type: "TAB_CAPTURE_COMPLETE",
                    requestId: message.requestId,
                    dataUrl: null,
                    error: null
                };
                if (chrome.runtime.lastError) {
                    console.error(`Background: Error capturing visible tab in windowId ${windowToCapture} (inspectedTabId ${message.tabId}):`, chrome.runtime.lastError.message);
                    response.error = chrome.runtime.lastError.message;
                } else if (!dataUrl) {
                    console.error(`Background: captureVisibleTab returned empty dataUrl for windowId ${windowToCapture} (inspectedTabId ${message.tabId}).`);
                    response.error = "captureVisibleTab returned empty dataUrl.";
                } else {
                    // console.log(`Background: Successfully captured tab in windowId ${windowToCapture}`); // Verbose
                    response.dataUrl = dataUrl;
                }
                if (currentConnection.panelPort) {
                    currentConnection.panelPort.postMessage(response);
                }
            });
        });
      }
    });

    port.onDisconnect.addListener(function() {
      console.log(`Background: DevTools panel disconnected for tab ${tabIdFromPortName}.`);
      const connInfo = activeWebSockets[tabIdFromPortName];
      if (connInfo) {
        connInfo.panelPort = null; 
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
    delete activeWebSockets[tabId];
  }
});

chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
  if (changeInfo.status === 'loading' && changeInfo.url) {
    if (activeWebSockets[tabId] && activeWebSockets[tabId].ws) {
      console.log(`Background: Tab ${tabId} navigated to ${changeInfo.url}. Closing WebSocket.`);
      activeWebSockets[tabId].ws.close();
    }
  }
});

console.log("Background service worker started.");
