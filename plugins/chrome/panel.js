const connectionNameInput = document.getElementById('connectionName');
const connectButton = document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null;

// Function to log to the inspected page's console
function logToInspectedPage(data) {
  const jsonStringData = JSON.stringify(data);
  const evalString = `console.log('[AgentMSG]', ${jsonStringData});`;
  chrome.devtools.inspectedWindow.eval(
    evalString,
    function(result, isException) {
      if (isException) {
        console.error('Error logging to inspected window:', isException);
        statusDiv.textContent = 'Error logging. See DevTools console (of this panel) for details.';
      }
    }
  );
}

function connect() {
  if (!panelPort) {
    // Establish a long-lived connection with the background script
    panelPort = chrome.runtime.connect({ name: "devtools-panel-" + chrome.devtools.inspectedWindow.tabId });
    console.log("Panel: Port connected to background script.");

    // Listen for messages from the background script
    panelPort.onMessage.addListener(function(message) {
      console.log("Panel: Received message from background:", message);
      if (message.type === "WEBSOCKET_MESSAGE") {
        logToInspectedPage(message.data);
        // Optionally, display the message in the panel too
        // const logEntry = document.createElement('div');
        // logEntry.textContent = JSON.stringify(message.data);
        // document.body.appendChild(logEntry);
      } else if (message.type === "WEBSOCKET_STATUS") {
        statusDiv.textContent = `Status: ${message.status}`;
        if (message.status === "Connected") {
          connectButton.style.display = 'none';
          disconnectButton.style.display = 'inline-block';
          connectionNameInput.disabled = true;
        } else {
          connectButton.style.display = 'inline-block';
          disconnectButton.style.display = 'none';
          connectionNameInput.disabled = false;
          currentConnectionName = null;
        }
      }
    });

    // Handle disconnection from the background script
    panelPort.onDisconnect.addListener(function() {
      console.warn("Panel: Port disconnected from background script.");
      statusDiv.textContent = "Status: Disconnected from background. Reload DevTools?";
      panelPort = null;
      connectButton.style.display = 'inline-block';
      disconnectButton.style.display = 'none';
      connectionNameInput.disabled = false;
    });
  }

  const connectionName = connectionNameInput.value.trim();
  if (connectionName) {
    currentConnectionName = connectionName;
    console.log(`Panel: Attempting to connect WebSocket for name: ${connectionName} on tab ${chrome.devtools.inspectedWindow.tabId}`);
    panelPort.postMessage({
      type: "CONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId, // Send tabId for clarity, though background can also get it
      connectionName: connectionName
    });
    statusDiv.textContent = "Status: Connecting...";
  } else {
    statusDiv.textContent = "Status: Please enter a connection name.";
  }
}

function disconnect() {
  if (panelPort && currentConnectionName) {
    console.log(`Panel: Sending disconnect request for ${currentConnectionName}`);
    panelPort.postMessage({
      type: "DISCONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId,
      connectionName: currentConnectionName // Send for verification if needed
    });
    // UI will be updated by WEBSOCKET_STATUS message from background
  } else {
    console.warn("Panel: No active connection or port to disconnect.");
     // Reset UI just in case
    statusDiv.textContent = "Status: Not Connected";
    connectButton.style.display = 'inline-block';
    disconnectButton.style.display = 'none';
    connectionNameInput.disabled = false;
  }
}

connectButton.addEventListener('click', connect);
disconnectButton.addEventListener('click', disconnect);

// Optional: Try to restore state if panel is reopened for same tab
// This is a bit more complex as background might not know panel reopened vs. a new one.
// For now, we rely on user clicking connect.
// chrome.storage.local.get([`lastConnectionName_${chrome.devtools.inspectedWindow.tabId}`], function(result) {
//   const lastName = result[`lastConnectionName_${chrome.devtools.inspectedWindow.tabId}`];
//   if (lastName) {
//     connectionNameInput.value = lastName;
//     // You might also want to ask background script if a connection for this name/tab is already active
//   }
// });

console.log("Panel.js loaded for tab: " + chrome.devtools.inspectedWindow.tabId);
