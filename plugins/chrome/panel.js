const connectionNameInput = document.getElementById('connectionName');
const connectButton = document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null;

// Function to log any data to the inspected page's console
function logToInspectedPageConsole(data, prefix = '[AgentMSG]') {
  const jsonStringData = JSON.stringify(data);
  const evalString = `console.log('${prefix}', ${jsonStringData});`;
  chrome.devtools.inspectedWindow.eval(
    evalString,
    function(result, isException) {
      if (isException) {
        console.error('Error logging to inspected window console:', isException);
        // statusDiv.textContent = 'Error logging. See DevTools console (of this panel) for details.';
      }
    }
  );
}

// Function to execute specific commands in the inspected page
function executeInInspectedPage(commandDetails) {
  let evalString = "";
  const commandType = commandDetails.type;

  console.log("Panel: Processing command: ", commandDetails);

  switch (commandType) {
    case 'SCROLL_TO_TOP':
      evalString = "window.scrollTo(0, 0); console.log('[AgentAction] Scrolled to top');";
      break;
    case 'SCROLL_TO_BOTTOM':
      evalString = "window.scrollTo(0, document.body.scrollHeight); console.log('[AgentAction] Scrolled to bottom');";
      break;
    case 'SCROLL_DELTA':
      const deltaY = commandDetails.deltaY || 0;
      const deltaX = commandDetails.deltaX || 0;
      evalString = `window.scrollBy(${deltaX}, ${deltaY}); console.log('[AgentAction] Scrolled by deltaX: ${deltaX}, deltaY: ${deltaY}');`;
      break;
    case 'QUERY_SELECTOR_ALL':
      if (!commandDetails.selector) {
        logToInspectedPageConsole({ error: "Selector missing for QUERY_SELECTOR_ALL", details: commandDetails }, "[AgentActionError]");
        return;
      }
      // This eval will return data, which will be handled by the callback.
      // We also log the initiation of the query.
      evalString = `
        console.log('[AgentAction] Executing querySelectorAll for: ${commandDetails.selector}');
        Array.from(document.querySelectorAll(${JSON.stringify(commandDetails.selector)})).map(el => {
          let details = {
            tagName: el.tagName,
            id: el.id,
            className: el.className,
            rect: el.getBoundingClientRect ? el.getBoundingClientRect() : null,
            innerText: el.innerText ? el.innerText.substring(0, 100) : null, // First 100 chars
            outerHTML: el.outerHTML ? el.outerHTML.substring(0, 200) : null // First 200 chars of outerHTML
          };
          return details;
        });
      `;
      // The result of this eval will be handled in the callback below.
      break;
    case 'SET_INPUT_VALUE':
      if (!commandDetails.selector || typeof commandDetails.value === 'undefined') {
        logToInspectedPageConsole({ error: "Selector or value missing for SET_INPUT_VALUE", details: commandDetails }, "[AgentActionError]");
        return;
      }
      evalString = `
        (() => {
          const el = document.querySelector(${JSON.stringify(commandDetails.selector)});
          if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
            el.value = ${JSON.stringify(commandDetails.value)};
            console.log('[AgentAction] Set value for selector: ${commandDetails.selector}');
            // Optionally, dispatch input/change events if needed for some frameworks
            // el.dispatchEvent(new Event('input', { bubbles: true }));
            // el.dispatchEvent(new Event('change', { bubbles: true }));
            return { success: true, selector: ${JSON.stringify(commandDetails.selector)}, valueSet: ${JSON.stringify(commandDetails.value)} };
          } else {
            console.error('[AgentActionError] Could not find input/textarea for selector: ${commandDetails.selector}');
            return { success: false, selector: ${JSON.stringify(commandDetails.selector)}, error: 'Element not found or not an input/textarea' };
          }
        })();
      `;
      // Result will be logged by the eval's callback
      break;
    default:
      // If it's not a known command, just log the raw data as before
      logToInspectedPageConsole(commandDetails, '[AgentMSG_Unknown]');
      return; // exit early
  }

  if (evalString) {
    chrome.devtools.inspectedWindow.eval(
      evalString,
      function(result, isException) {
        if (isException) {
          console.error(`Error executing command ${commandType} in inspected window:`, isException);
          logToInspectedPageConsole({ error: `Failed to execute ${commandType}`, details: isException }, "[AgentActionError]");
          statusDiv.textContent = `Error for ${commandType}. See panel console.`;
        } else {
          console.log(`Result of ${commandType}:`, result);
          // For QUERY_SELECTOR_ALL, the result IS the data we want to log.
          // For other commands, result might be undefined or a simple confirmation.
          if (commandType === 'QUERY_SELECTOR_ALL') {
            logToInspectedPageConsole({ query: commandDetails.selector, results: result, requestId: commandDetails.requestId }, "[AgentQueryResult]");
          } else if (commandType === 'SET_INPUT_VALUE') {
             logToInspectedPageConsole(result, "[AgentActionReport]"); // Log success/failure object
          }
          // No need for generic status update here unless specific error
        }
      }
    );
  }
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

  const connectionNameVal = connectionNameInput.value.trim(); // Renamed to avoid conflict
  if (connectionNameVal) {
    currentConnectionName = connectionNameVal;
    console.log(`Panel: Attempting to connect WebSocket for name: ${currentConnectionName} on tab ${chrome.devtools.inspectedWindow.tabId}`);
    panelPort.postMessage({
      type: "CONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId,
      connectionName: currentConnectionName
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
      connectionName: currentConnectionName
    });
  } else {
    console.warn("Panel: No active connection or port to disconnect.");
    statusDiv.textContent = "Status: Not Connected";
    connectButton.style.display = 'inline-block';
    disconnectButton.style.display = 'none';
    connectionNameInput.disabled = false;
  }
}

connectButton.addEventListener('click', connect);
disconnectButton.addEventListener('click', disconnect);

console.log("Panel.js loaded for tab: " + chrome.devtools.inspectedWindow.tabId);
