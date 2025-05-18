const connectionNameInput =  document.getElementById('connectionName');
const connectButton = document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null;

// Function to log any data to the inspected page's console
function logToInspectedPageConsole(data, prefix = '[AgentMSG]') {
  const jsonStringData = JSON.stringify(data);
  // Ensure prefix is a string literal in the eval'd code
  const evalString = `console.log(${JSON.stringify(prefix)}, ${jsonStringData});`;
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
  // Ensure commandDetails and its type property exist
  if (!commandDetails || typeof commandDetails.type === 'undefined') {
    logToInspectedPageConsole({ error: "Command details or type is missing", received: commandDetails }, "[AgentActionError]");
    return;
  }
  const commandType = commandDetails.type;

  console.log("Panel: Processing command: ", commandDetails);

  switch (commandType) {
    case 'SCROLL_TO_TOP':
      // Using your custom scroll target if present, otherwise window
      evalString = `
        (() => {
          const scroller = document.querySelector("ms-autoscroll-container") || window;
          scroller.scrollTo(0, 0);
          console.log('[AgentAction] Scrolled to top');
        })();
      `;
      break;
    case 'SCROLL_TO_BOTTOM':
      // Using your custom scroll target
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
      // Using your custom scroll target
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
            rect: el.getBoundingClientRect ? el.getBoundingClientRect() : {}, // Ensure rect is an object
            innerText: el.innerText ? el.innerText.substring(0, 100) : '',    // Ensure string
            outerHTML: el.outerHTML ? el.outerHTML.substring(0, 200) : ''  // Ensure string
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
      evalString = `
        (() => {
          const el = document.querySelector(${JSON.stringify(commandDetails.selector)});
          const submitSelector = (${JSON.stringify(commandDetails.submitSelector)} || "").trim();
          const submitButton = (submitSelector.length > 0) ? document.querySelector(submitSelector) : null;

          if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
            el.value = ${JSON.stringify(commandDetails.value)};
            console.log('[AgentAction] Set value for selector: ' + ${JSON.stringify(commandDetails.selector)});
            // el.dispatchEvent(new Event('input', { bubbles: true })); // Uncomment if needed
            // el.dispatchEvent(new Event('change', { bubbles: true }));// Uncomment if needed

            return { success: true, selector: ${JSON.stringify(commandDetails.selector)}, valueSet: ${JSON.stringify(commandDetails.value)} };
          } else {
            console.error('[AgentActionError] Could not find input/textarea for selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: false, selector: ${JSON.stringify(commandDetails.selector)}, error: 'Element not found or not an input/textarea' };
          }
        })();
      `;
      break;
    case 'CLICK_ELEMENT': // New case
      if (!commandDetails.selector) {
        logToInspectedPageConsole({ error: "Selector missing for CLICK_ELEMENT", details: commandDetails }, "[AgentActionError]");
        return;
      }
      evalString = `
        (() => {
          const el = document.querySelector(${JSON.stringify(commandDetails.selector)});
          if (el && typeof el.click === 'function') {
            el.click();
            console.log('[AgentAction] Clicked element with selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: true, selector: ${JSON.stringify(commandDetails.selector)} };
          } else {
            console.error('[AgentActionError] Could not find clickable element for selector: ' + ${JSON.stringify(commandDetails.selector)});
            return { success: false, selector: ${JSON.stringify(commandDetails.selector)}, error: 'Element not found or not clickable' };
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
          } else if (commandType === 'SET_INPUT_VALUE' || commandType === 'CLICK_ELEMENT') { // Updated to include CLICK_ELEMENT
             logToInspectedPageConsole(result, "[AgentActionReport]");
          }
          // For other commands, the console.log within their evalString serves as confirmation.
        }
      }
    );
  }
}


function connect() {
  if (!panelPort) {
    panelPort = chrome.runtime.connect({ name: "devtools-panel-" + chrome.devtools.inspectedWindow.tabId });
    console.log("Panel: Port connected to background script.");

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
        } else {
          if (connectButton) connectButton.style.display = 'inline-block';
          if (disconnectButton) disconnectButton.style.display = 'none';
          if (connectionNameInput) connectionNameInput.disabled = false;
          currentConnectionName = null;
        }
      }
    });

    panelPort.onDisconnect.addListener(function() {
      console.warn("Panel: Port disconnected from background script.");
      if (statusDiv) statusDiv.textContent = "Status: Disconnected from background. Reload DevTools?";
      panelPort = null;
      if (connectButton) connectButton.style.display = 'inline-block';
      if (disconnectButton) disconnectButton.style.display = 'none';
      if (connectionNameInput) connectionNameInput.disabled = false;
    });
  }

  const connectionNameVal = connectionNameInput ? connectionNameInput.value.trim() : "";
  if (connectionNameVal) {
    currentConnectionName = connectionNameVal;
    console.log(`Panel: Attempting to connect WebSocket for name: ${currentConnectionName} on tab ${chrome.devtools.inspectedWindow.tabId}`);
    panelPort.postMessage({
      type: "CONNECT_WEBSOCKET",
      tabId: chrome.devtools.inspectedWindow.tabId,
      connectionName: currentConnectionName
    });
    if (statusDiv) statusDiv.textContent = "Status: Connecting...";
  } else {
    if (statusDiv) statusDiv.textContent = "Status: Please enter a connection name.";
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
    if (statusDiv) statusDiv.textContent = "Status: Not Connected";
    if (connectButton) connectButton.style.display = 'inline-block';
    if (disconnectButton) disconnectButton.style.display = 'none';
    if (connectionNameInput) connectionNameInput.disabled = false;
  }
}

if (connectButton) connectButton.addEventListener('click', connect);
if (disconnectButton) disconnectButton.addEventListener('click', disconnect);

console.log("Panel.js loaded for tab: " + chrome.devtools.inspectedWindow.tabId);
