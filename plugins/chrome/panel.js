const connectionNameInput =  document.getElementById('connectionName');
const connectButton =  document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null; // Store the name used for the current/last successful connection

// Reconnection logic variables
let reconnectAttempts = 0;
let reconnectTimeoutId = null;
let userInitiatedDisconnect = false; // Flag to prevent auto-reconnect if user clicks "Disconnect"
const INITIAL_RECONNECT_DELAY = 1000; // 1 second
const MAX_RECONNECT_DELAY = 10000;    // 10 seconds

// Function to log any data to the inspected page's console
function logToInspectedPageConsole(data, prefix = '[AgentMSG]') {
  const jsonStringData = JSON.stringify(data);
  const evalString = `console.log(${JSON.stringify(prefix)}, ${jsonStringData});`;
  chrome.devtools.inspectedWindow.eval(
    evalString,
    function(result, isException) {
      if (isException) {
        console.error('Error logging to inspected window console:', isException);
      }
    }
  );
}

// Function to execute specific commands in the inspected page
function executeInInspectedPage(commandDetails) {
  let evalString = "";
  if (!commandDetails || typeof commandDetails.type === 'undefined') {
    logToInspectedPageConsole({ error: "Command details or type is missing", received: commandDetails }, "[AgentActionError]");
    return;
  }
  const commandType = commandDetails.type;
  console.log("Panel: Processing command: ", commandDetails);

  switch (commandType) {
    case 'SCROLL_TO_TOP':
      evalString = `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || window;
          scroller.scrollTo(0, 0);
          console.log('[AgentAction] Scrolled to top');
        })();
      `;
      break;
    case 'SCROLL_TO_BOTTOM':
      evalString = `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || document.body;
          let target = scroller === window ? document.body.scrollHeight : scroller.scrollHeight;
          (scroller === window ? window : scroller).scrollTo(0, target);
          console.log('[AgentAction] Scrolled to bottom');
        })();
      `;
      break;
    case 'SCROLL_DELTA':
      const deltaY = commandDetails.deltaY || 0;
      const deltaX = commandDetails.deltaX || 0;
      evalString = `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || window;
          (scroller === window ? window : scroller).scrollBy(${deltaX}, ${deltaY});
          console.log('[AgentAction] Scrolled by deltaX: ${deltaX}, deltaY: ${deltaY}');
        })();
      `;
      break;
    case 'QUERY_SELECTOR_ALL':
      if (!commandDetails.selector) {
        logToInspectedPageConsole({ error: "Selector missing for QUERY_SELECTOR_ALL", details: commandDetails }, "[AgentActionError]");
        return;
      }
      evalString = `
        console.log('[AgentAction] Executing querySelectorAll for: ' + ${JSON.stringify(commandDetails.selector)});
        Array.from(document.querySelectorAll(${JSON.stringify(commandDetails.selector)})).map(el => {
          let details = {
            tagName: el.tagName,
            id: el.id,
            className: el.className,
            rect: el.getBoundingClientRect ? el.getBoundingClientRect() : {},
            innerText: el.innerText ? el.innerText.substring(0, 100) : '',
            outerHTML: el.outerHTML ? el.outerHTML.substring(0, 200) : ''
          };
          return details;
        });
      `;
      break;
    case 'SET_INPUT_VALUE':
      if (!commandDetails.selector || typeof commandDetails.value === 'undefined') {
        logToInspectedPageConsole({ error: "Selector or value missing for SET_INPUT_VALUE", details: commandDetails }, "[AgentActionError]");
        return;
      }
      // Including your submitSelector logic
      const shouldSubmit = JSON.stringify(commandDetails.submit); 
      const submitSelector = JSON.stringify((commandDetails.submitSelector || "").trim());
      evalString = `
        (() => {
          const el = document.querySelector(${JSON.stringify(commandDetails.selector)});
          if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
            el.focus();
            el.value = ${JSON.stringify(commandDetails.value)};
            el.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
            el.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
            console.log('[AgentAction] Set value and dispatched events for selector: ' + ${JSON.stringify(commandDetails.selector)});
            
            if (${shouldSubmit} && ${submitSelector} !== '""') {
                const submitBtn = document.querySelector(${submitSelector});
                if (submitBtn instanceof HTMLElement) {
                    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window });
                    submitBtn.dispatchEvent(clickEvent);
                    console.log('[AgentAction] Also clicked submit button: ' + ${submitSelector});
                } else {
                    console.warn('[AgentAction] Submit button not found for selector: ' + ${submitSelector});
                }
            }
            return { success: true, selector: ${JSON.stringify(commandDetails.selector)}, valueSet: ${JSON.stringify(commandDetails.value)} };
          } else {
            console.error('[AgentActionError] Could not find input/textarea for selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: false, selector: ${JSON.stringify(commandDetails.selector)}, error: 'Element not found or not an input/textarea' };
          }
        })();
      `;
      break;
    case 'CLICK_ELEMENT':
      if (!commandDetails.selector) {
        logToInspectedPageConsole({ error: "Selector missing for CLICK_ELEMENT", details: commandDetails }, "[AgentActionError]");
        return;
      }
      evalString = `
        (() => {
          const el = document.querySelector(${JSON.stringify(commandDetails.selector)});
          if (el instanceof HTMLElement) {
            const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window });
            el.dispatchEvent(clickEvent);
            console.log('[AgentAction] Dispatched click event on element with selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: true, selector: ${JSON.stringify(commandDetails.selector)} };
          } else {
            console.error('[AgentActionError] Could not find HTMLElement for click selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: false, selector: ${JSON.stringify(commandDetails.selector)}, error: 'Element not found or not an HTMLElement' };
          }
        })();
      `;
      break;
    default:
      logToInspectedPageConsole(commandDetails, '[AgentMSG_UnknownType]');
      return; 
  }

  if (evalString) {
    chrome.devtools.inspectedWindow.eval(
      evalString,
      function(result, isException) {
        if (isException) {
          console.error(`Error executing command ${commandType} in inspected window:`, isException);
          logToInspectedPageConsole({ error: `Failed to execute ${commandType}`, details: isException }, "[AgentActionError]");
          if (statusDiv) statusDiv.textContent = `Error for ${commandType}. See panel console.`;
        } else {
          console.log(`Result of ${commandType}:`, result);
          if (commandType === 'QUERY_SELECTOR_ALL') {
            logToInspectedPageConsole({ query: commandDetails.selector, results: result, requestId: commandDetails.requestId }, "[AgentQueryResult]");
          } else if (commandType === 'SET_INPUT_VALUE' || commandType === 'CLICK_ELEMENT') {
             logToInspectedPageConsole(result, "[AgentActionReport]");
          }
        }
      }
    );
  }
}

function scheduleReconnect() {
    if (userInitiatedDisconnect) {
        console.log("Panel: User initiated disconnect. Not attempting auto-reconnect.");
        return;
    }
    if (!currentConnectionName) {
        console.log("Panel: No current connection name available. Cannot auto-reconnect.");
        if (statusDiv) statusDiv.textContent = "Status: Disconnected. Enter name and connect.";
        if (connectButton) connectButton.style.display = 'inline-block';
        if (disconnectButton) disconnectButton.style.display = 'none';
        if (connectionNameInput) connectionNameInput.disabled = false;
        return;
    }

    reconnectAttempts++;
    let delay = INITIAL_RECONNECT_DELAY * Math.pow(2, reconnectAttempts -1 );
    if (delay > MAX_RECONNECT_DELAY) {
        delay = MAX_RECONNECT_DELAY;
    }

    console.log(`Panel: Attempting reconnect #${reconnectAttempts} in ${delay / 1000} seconds for ${currentConnectionName}.`);
    if (statusDiv) statusDiv.textContent = `Status: Disconnected. Reconnecting in ${Math.round(delay/1000)}s... (Attempt ${reconnectAttempts})`;
    
    clearTimeout(reconnectTimeoutId); // Clear any existing timeout
    reconnectTimeoutId = setTimeout(() => {
        console.log("Panel: Reconnect timer fired. Calling connect().");
        // Ensure connectionNameInput has the currentConnectionName for the connect() function to use
        if (connectionNameInput) connectionNameInput.value = currentConnectionName; 
        connect(false); // Pass flag to indicate it's not a direct user click
    }, delay);
}

function clearReconnectTimer() {
    clearTimeout(reconnectTimeoutId);
    reconnectTimeoutId = null;
}

function connect(isUserClick = true) {
  clearReconnectTimer(); // Always clear pending reconnects when attempting to connect
  if (isUserClick) {
    userInitiatedDisconnect = false; // User wants to connect/reconnect
    reconnectAttempts = 0; // Reset attempts on manual connect
  }

  if (!panelPort) { // Only establish port if it doesn't exist
    panelPort = chrome.runtime.connect({ name: "devtools-panel-" + chrome.devtools.inspectedWindow.tabId });
    console.log("Panel: Port (re)established with background script.");

    panelPort.onMessage.addListener(function(message) {
      console.log("Panel: Received message from background:", message);
      if (message.type === "WEBSOCKET_MESSAGE") {
        if (message.data) {
            executeInInspectedPage(message.data);
        } else {
            logToInspectedPageConsole({error: "Received empty data in WEBSOCKET_MESSAGE"}, "[AgentMSGError]");
        }
      } else if (message.type === "WEBSOCKET_STATUS") {
        if (statusDiv) statusDiv.textContent = `Status: ${message.status}`; 
        if (message.status === "Connected") {
          if (connectButton) connectButton.style.display = 'none';
          if (disconnectButton) disconnectButton.style.display = 'inline-block';
          if (connectionNameInput) connectionNameInput.disabled = true;
          userInitiatedDisconnect = false; // Reset on successful connect
          reconnectAttempts = 0; // Reset on successful connect
          clearReconnectTimer(); // Clear any reconnect timer as we are now connected
        } else { // Disconnected or Error status
          if (connectButton) connectButton.style.display = 'inline-block';
          if (disconnectButton) disconnectButton.style.display = 'none';
          if (connectionNameInput) connectionNameInput.disabled = false;
          
          // Only schedule reconnect if it wasn't a user-initiated disconnect
          // and the status indicates a closed/errored WebSocket
          if (!userInitiatedDisconnect && (message.status === "Disconnected" || message.status.startsWith("Error:"))) {
             console.log("Panel: WebSocket status is not 'Connected'. Scheduling reconnect.");
             scheduleReconnect();
          } else if (userInitiatedDisconnect) {
             console.log("Panel: WebSocket status is not 'Connected', but user initiated disconnect. No auto-reconnect.");
             currentConnectionName = null; // Clear connection name as user disconnected
          }
        }
      }
    });

    panelPort.onDisconnect.addListener(function() {
      console.warn("Panel: Port to background script disconnected.");
      if (statusDiv) statusDiv.textContent = "Status: Disconnected from background script.";
      panelPort = null; // Nullify the port
      // Don't immediately try to reconnect the port itself here,
      // rely on WebSocket status or user action.
      // If WebSocket was active, its closure should trigger scheduleReconnect.
      // If user tries to connect, a new port will be made.
      if (connectButton) connectButton.style.display = 'inline-block';
      if (disconnectButton) disconnectButton.style.display = 'none';
      if (connectionNameInput) connectionNameInput.disabled = false;
      
      // If the port disconnects, and we weren't told the WebSocket closed,
      // it might be an extension reload or something.
      // We could try to schedule a reconnect for the WebSocket if a connection was active.
      if (!userInitiatedDisconnect && currentConnectionName) {
          console.log("Panel: Port disconnected, attempting to re-establish WebSocket connection.");
          scheduleReconnect();
      }
    });
  }

  const nameToConnect = connectionNameInput ? connectionNameInput.value.trim() : "";
  if (nameToConnect) {
    currentConnectionName = nameToConnect; // Update currentConnectionName for reconnects
    console.log(`Panel: Attempting to connect WebSocket for name: ${currentConnectionName} on tab ${chrome.devtools.inspectedWindow.tabId}`);
    if (panelPort) { // Ensure port is available before posting message
        panelPort.postMessage({
          type: "CONNECT_WEBSOCKET",
          tabId: chrome.devtools.inspectedWindow.tabId,
          connectionName: currentConnectionName
        });
        if (statusDiv && !isUserClick && reconnectAttempts > 0) {
            // Status is already "Reconnecting...", don't overwrite with "Connecting..."
        } else if (statusDiv) {
            statusDiv.textContent = "Status: Connecting...";
        }
    } else {
        console.error("Panel: panelPort is null, cannot send CONNECT_WEBSOCKET message. This usually means the background script connection was lost.");
        if(statusDiv) statusDiv.textContent = "Status: Error - No connection to background script.";
        // Potentially try to re-establish panelPort here or guide user
    }
  } else {
    if (isUserClick && statusDiv) statusDiv.textContent = "Status: Please enter a connection name.";
    // Don't clear currentConnectionName if it's an auto-reconnect attempt without input value
  }
}

function disconnect() {
  userInitiatedDisconnect = true; // User clicked disconnect
  clearReconnectTimer(); // Stop any attempts to reconnect
  reconnectAttempts = 0;

  if (panelPort && currentConnectionName) {
    console.log(`Panel: User initiated disconnect for ${currentConnectionName}`);
    panelPort.postMessage({
      type: "DISCONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId,
      connectionName: currentConnectionName 
    });
    // UI update will happen via WEBSOCKET_STATUS message ("Disconnected")
  } else {
    console.warn("Panel: No active connection or port to disconnect.");
    if (statusDiv) statusDiv.textContent = "Status: Not Connected";
    if (connectButton) connectButton.style.display = 'inline-block';
    if (disconnectButton) disconnectButton.style.display = 'none';
    if (connectionNameInput) connectionNameInput.disabled = false;
  }
  // currentConnectionName = null; // Clear after sending disconnect
}

if (connectButton) connectButton.addEventListener('click', () => connect(true));
if (disconnectButton) disconnectButton.addEventListener('click', disconnect);

console.log("Panel.js loaded for tab: " + chrome.devtools.inspectedWindow.tabId);

// Attempt to restore connection name from session storage if panel is reloaded for the same tab
// This is a simple way to remember the last name for this tab session.
const storageKey = `lastConnectionName_${chrome.devtools.inspectedWindow.tabId}`;
const savedName = sessionStorage.getItem(storageKey);
if (savedName && connectionNameInput) {
    connectionNameInput.value = savedName;
    // Optionally, you could try to query background if a connection is already active for this.
    // For now, user still needs to click connect.
}

// Save connection name on successful connect or user input change
if (connectionNameInput) {
    connectionNameInput.addEventListener('change', (event) => {
        if (event.target.value) {
            sessionStorage.setItem(storageKey, event.target.value);
        } else {
            sessionStorage.removeItem(storageKey);
        }
    });
}
