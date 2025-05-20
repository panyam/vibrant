const connectionNameInput =  document.getElementById('connectionName');
const connectButton =  document.getElementById('connectButton');
const disconnectButton = document.getElementById('disconnectButton');
const statusDiv = document.getElementById('status');

let panelPort = null;
let currentConnectionName = null;

let reconnectAttempts = 0;
let reconnectTimeoutId = null;
let userInitiatedDisconnect = false;
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 10000;

// Store pending screenshot callbacks by requestId
const pendingScreenshotCallbacks = {};

function sendResponseToBackend(payload) {
    if (panelPort) {
        panelPort.postMessage({
            type: "FORWARD_TO_WEBSOCKET_SERVER",
            payload: payload
        });
    } else {
        console.error("Panel: panelPort is null, cannot send response to backend.", payload);
    }
}

function handleEvaluateScriptRequest(requestDetails) {
    const scriptToEvaluate = requestDetails.payload;
    if (!requestDetails || !requestDetails.requestId || typeof scriptToEvaluate !== 'string') {
        console.error("Panel: Invalid EVALUATE_SCRIPT request", requestDetails);
        sendResponseToBackend({
            type: "EVALUATION_RESULT",
            requestId: requestDetails.requestId || "unknown",
            isException: true,
            exceptionInfo: "Invalid EVALUATE_SCRIPT request structure from backend"
        });
        return;
    }

    chrome.devtools.inspectedWindow.eval(
        scriptToEvaluate,
        function(result, isExceptionInfo) {
            let exceptionDetails = null;
            if (isExceptionInfo) {
                if (typeof isExceptionInfo === 'object' && isExceptionInfo !== null) {
                    exceptionDetails = isExceptionInfo.description || isExceptionInfo.value || JSON.stringify(isExceptionInfo);
                } else {
                    exceptionDetails = String(isExceptionInfo);
                }
                 console.error(`Panel: Exception during script evaluation (ReqID: ${requestDetails.requestId}):`, exceptionDetails, "Original script:", scriptToEvaluate);
            }
            sendResponseToBackend({
                type: "EVALUATION_RESULT",
                requestId: requestDetails.requestId,
                result: result,
                isException: !!isExceptionInfo,
                exceptionInfo: exceptionDetails
            });
        }
    );
}

async function handleCaptureElementsScreenshotRequest(requestDetails) {
    const requestId = requestDetails.requestId;
    const selectors = requestDetails.payload;
    if (!requestId || !Array.isArray(selectors) || selectors.length === 0) {
        console.error("Panel: Invalid CAPTURE_ELEMENTS_SCREENSHOT request", requestDetails);
        sendResponseToBackend({
            type: "ELEMENTS_SCREENSHOT_RESULT",
            requestId: requestId || "unknown_screenshot_request",
            imageData: {},
            error: "Invalid CAPTURE_ELEMENTS_SCREENSHOT request structure or empty selectors."
        });
        return;
    }

    console.log(`Panel: Starting screenshot capture for requestId: ${requestId}, selectors:`, selectors);

    const getElementDataScript = `
        (() => {
            const selectors = ${JSON.stringify(selectors)};
            const results = {};
            const dpr = window.devicePixelRatio || 1;
            selectors.forEach(selector => {
                const element = document.querySelector(selector);
                if (element) {
                    const rect = element.getBoundingClientRect();
                    results[selector] = {
                        x: rect.x, y: rect.y, width: rect.width, height: rect.height,
                        top: rect.top, left: rect.left, devicePixelRatio: dpr
                    };
                } else {
                    results[selector] = null;
                }
            });
            return results;
        })();
    `;

    try {
        const elementDataMap = await new Promise((resolve, reject) => {
            chrome.devtools.inspectedWindow.eval(getElementDataScript, (result, isException) => {
                if (isException) {
                    console.error('Panel: Error getting element data:', isException);
                    reject(isException.description || isException.value || JSON.stringify(isException));
                } else {
                    resolve(result);
                }
            });
        });

        if (!elementDataMap) throw new Error("Failed to retrieve element data from page.");
        console.log(`Panel: Got element data for ${requestId}:`, elementDataMap);

        const currentTabId = chrome.devtools.inspectedWindow.tabId;
        console.log(`Panel: Requesting tab capture from background for tab ID: ${currentTabId}, requestId: ${requestId}`);

        // Create a promise that will be resolved when the background script sends back the capture
        const capturePromise = new Promise((resolve, reject) => {
            pendingScreenshotCallbacks[requestId] = { resolve, reject };
        });

        if (panelPort) {
            panelPort.postMessage({
                type: "REQUEST_TAB_CAPTURE",
                tabId: currentTabId,
                requestId: requestId
            });
        } else {
            throw new Error("Panel port not connected, cannot request tab capture.");
        }

        // Wait for the background script to send back the dataUrl or an error
        const { dataUrl, error: captureError } = await capturePromise;
        // Clean up callback
        delete pendingScreenshotCallbacks[requestId];

        if (captureError) throw new Error(captureError);
        if (!dataUrl) throw new Error("Background script did not return a dataUrl for tab capture.");

        console.log(`Panel: Received full page dataUrl from background for ${requestId}. Processing crops...`);

        const mainImage = new Image();
        const imageDataResults = {};

        await new Promise((resolveImageLoad, rejectImageLoad) => {
            mainImage.onload = () => {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                for (const selector in elementDataMap) {
                    const data = elementDataMap[selector];
                    if (data && data.width > 0 && data.height > 0) {
                        const dpr = data.devicePixelRatio;
                        canvas.width = Math.round(data.width * dpr);
                        canvas.height = Math.round(data.height * dpr);
                        ctx.drawImage(mainImage,
                            Math.round(data.left * dpr), Math.round(data.top * dpr),
                            Math.round(data.width * dpr), Math.round(data.height * dpr),
                            0, 0,
                            Math.round(data.width * dpr), Math.round(data.height * dpr));
                        imageDataResults[selector] = canvas.toDataURL("image/png");
                    } else {
                        imageDataResults[selector] = null;
                    }
                }
                resolveImageLoad();
            };
            mainImage.onerror = (err) => {
                 console.error('Panel: Failed to load main screenshot image:', err);
                 rejectImageLoad("Failed to load main screenshot image for cropping.");
            };
            mainImage.src = dataUrl;
        });

        console.log(`Panel: Finished cropping for ${requestId}. Sending results.`);
        sendResponseToBackend({
            type: "ELEMENTS_SCREENSHOT_RESULT",
            requestId: requestId,
            imageData: imageDataResults,
            error: null
        });

    } catch (error) {
        console.error(`Panel: Error during screenshot process for ${requestId}:`, error);
        const errorMessage = (typeof error === 'string' ? error : (error && error.message)) || "Unknown error in screenshot capture";
        sendResponseToBackend({
            type: "ELEMENTS_SCREENSHOT_RESULT",
            requestId: requestId,
            imageData: {},
            error: errorMessage
        });
         // Clean up callback if an error occurred before it was resolved
        if (pendingScreenshotCallbacks[requestId]) {
            delete pendingScreenshotCallbacks[requestId];
        }
    }
}

async function handlePasteDataRequest(requestDetails) {
    const { requestId, payload } = requestDetails;
    // payload is expected to be { selector: "...", dataUrl: "..." }
    const { selector, dataUrl } = payload;

    if (!requestId || !selector || !dataUrl) {
        console.error("Panel: Invalid PASTE_DATA request", requestDetails);
        sendResponseToBackend({
            type: "PASTE_RESULT",
            requestId: requestId || "unknown_paste_request",
            success: false,
            message: null,
            error: "Invalid PASTE_DATA request structure (missing requestId, selector, or dataUrl)."
        });
        return;
    }

    console.log(`Panel: Starting paste data for requestId: ${requestId}, selector: ${selector}`);

    // Construct the script to be evaluated in the inspected window
    // The selector and dataUrl are passed as arguments to an IIFE.
    // Single quotes within selector and dataUrl are escaped for the string literal arguments of the IIFE.
    const scriptToExecute = `
        (async (selectorArg, dataUrlArg) => {
            const element = document.querySelector(selectorArg);
            if (!element) {
                return { success: false, error: \`Element "\${selectorArg}" not found.\` };
            }
            element.focus();

            try {
                const fetchResponse = await fetch(dataUrlArg);
                if (!fetchResponse.ok) {
                    throw new Error(\`Failed to fetch data URL (status: \${fetchResponse.status}) starting with: \${dataUrlArg.substring(0,100)}...\`);
                }
                const blob = await fetchResponse.blob();

                const dataTransfer = new DataTransfer();
                const fileExtension = blob.type.split('/')[1] || 'png';
                const fileName = "pasted_image." + fileExtension;
                dataTransfer.items.add(new File([blob], fileName, { type: blob.type }));

                const pasteEvent = new ClipboardEvent('paste', {
                    clipboardData: dataTransfer,
                    bubbles: true,
                    cancelable: true
                });
                element.dispatchEvent(pasteEvent);

                return { success: true, message: \`Paste event dispatched to "\${selectorArg}".\` };

            } catch (e) {
                console.error(\`[Vibrant Agent] Error during paste simulation for selector "\${selectorArg}":\`, e);
                return { success: false, error: e.message || String(e) };
            }
        })(\`${selector.replace(/`/g, '\\\\`')}\`, \`${dataUrl.replace(/`/g, '\\\\`')}\`);
    `; // Using template literals for args, escaping backticks in them.

    chrome.devtools.inspectedWindow.eval(
        scriptToExecute,
        (result, isExceptionInfo) => {
            let responsePayload;
            if (isExceptionInfo) {
                const errorMsg = (isExceptionInfo && (isExceptionInfo.description || isExceptionInfo.value)) || JSON.stringify(isExceptionInfo);
                console.error(`Panel: Exception during paste script execution (ReqID: ${requestId}):`, errorMsg);
                responsePayload = {
                    type: "PASTE_RESULT",
                    requestId: requestId,
                    success: false,
                    message: null,
                    error: `Script evaluation exception: ${errorMsg}`
                };
            } else {
                // result should be {success: bool, message?: string, error?: string}
                responsePayload = {
                    type: "PASTE_RESULT",
                    requestId: requestId,
                    success: result.success,
                    message: result.message || null,
                    error: result.error || null
                };
            }
            sendResponseToBackend(responsePayload);
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
          if (message.type === "WEBSOCKET_MESSAGE") {
            if (message.data) {
                if (message.data.type === 'EVALUATE_SCRIPT') {
                    handleEvaluateScriptRequest(message.data);
                } else if (message.data.type === 'CAPTURE_ELEMENTS_SCREENSHOT') {
                    handleCaptureElementsScreenshotRequest(message.data);
                } else if (message.data.type === 'PASTE_DATA') { // Added this case
                    handlePasteDataRequest(message.data);      // Added this call
                } else {
                    console.warn("Panel: Received unhandled WebSocket message data type from server:", message.data.type, message.data);
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
          } else if (message.type === "TAB_CAPTURE_COMPLETE") {
            console.log("Panel: Received TAB_CAPTURE_COMPLETE from background:", message.requestId);
            const callback = pendingScreenshotCallbacks[message.requestId];
            if (callback) {
                if (message.error) {
                    callback.reject(message.error);
                } else {
                    callback.resolve({ dataUrl: message.dataUrl });
                }
            } else {
                console.warn("Panel: Received TAB_CAPTURE_COMPLETE for unknown requestId:", message.requestId);
            }
          } else {
            console.warn("Panel: Received unhandled message type directly from background:", message.type, message);
          }
        });

        panelPort.onDisconnect.addListener(function() {
          console.warn("Panel: Port to background script disconnected.");
          if (statusDiv) statusDiv.textContent = "Status: Disconnected from background script.";
          panelPort = null;
          for (const id in pendingScreenshotCallbacks) {
            pendingScreenshotCallbacks[id].reject("Panel port disconnected during screenshot process.");
            delete pendingScreenshotCallbacks[id];
          }
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
