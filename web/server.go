package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket" // Still needed by gohttp.JSONConn indirectly, or for types
	"github.com/panyam/goutils/conc"
	gohttp "github.com/panyam/goutils/http"
)

// Request struct remains the same
type Request struct {
	Id         string
	ClientId   string
	Payload    any // Could store the original script or params for context
	SentAt     time.Time
	Response   any // Stores the 'result' field from EVALUATION_RESULT or imageData for screenshots
	ResponseAt time.Time
	Timeout    time.Duration
	err        error       // Stores error if evaluation resulted in an exception
	finished   bool        // To prevent processing a response multiple times
	recvChan   chan string // For the /eval?wait=true or /screenshots?wait=true handler, made buffered
}

func NewRequest(clientId string, payload any) *Request {
	// Using a more specific prefix for screenshot requests for clarity in logs if needed
	prefix := "eval"
	if _, ok := payload.([]string); ok { // Check if payload is a slice of strings (selectors for screenshot)
		prefix = "screenshot"
	}
	return &Request{
		Id:       fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixNano(), clientId),
		SentAt:   time.Now(),
		ClientId: clientId,
		Payload:  payload,
		Timeout:  30 * time.Second,
		recvChan: make(chan string, 1), // Buffered channel of size 1
	}
}

// Json method of Request is not directly used for WebSocket message, but good for logging/internal state
func (r *Request) Json() any {
	return map[string]any{
		"Payload":   r.Payload,
		"RequestId": r.Id,
	}
}

type Handler struct {
	fanoutMutex     sync.RWMutex
	Fanouts         map[string]*conc.FanOut[conc.Message[any]]
	reqMutex        sync.RWMutex
	pendingRequests map[string]*Request
}

func NewHandler() *Handler {
	return &Handler{
		Fanouts:         make(map[string]*conc.FanOut[conc.Message[any]]),
		pendingRequests: make(map[string]*Request),
	}
}

type Conn struct {
	gohttp.JSONConn
	handler  *Handler
	ClientId string
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) (out *Conn, isValid bool) {
	out = &Conn{handler: h}
	out.ClientId = r.PathValue("clientId")
	if out.ClientId == "" {
		log.Println("Validation failed: clientId is missing in path")
		http.Error(w, "Client ID is missing in path", http.StatusBadRequest)
		return nil, false
	}
	log.Printf("Validated WebSocket connection for ClientId: %s", out.ClientId)
	isValid = true
	return
}

func (t *Conn) HandleMessage(msgData any) error {
	msgMap, ok := msgData.(map[string]any)
	if !ok {
		log.Printf("Client %s: Received message in unexpected format: %T. Data: %v", t.ClientId, msgData, msgData)
		return fmt.Errorf("unexpected message format: %T", msgData)
	}

	log.Printf("Client %s: Received WebSocket Message Data: %v", t.ClientId, msgMap)

	msgType, typeOk := msgMap["type"].(string)
	requestId, idOk := msgMap["requestId"].(string)

	if !typeOk || !idOk {
		log.Printf("Client %s: Received message without 'type' or 'requestId': %v", t.ClientId, msgMap)
		return fmt.Errorf("message from client missing 'type' or 'requestId'")
	}

	t.handler.reqMutex.Lock()
	defer t.handler.reqMutex.Unlock()

	req, exists := t.handler.pendingRequests[requestId]
	if !exists {
		log.Printf("Client %s: Received %s for unknown or already processed requestId %s.", t.ClientId, msgType, requestId)
		return nil // Don't return error, just log and ignore if request is not found
	}

	if req.finished {
		log.Printf("Client %s: Received duplicate/late %s for already finished requestId %s.", t.ClientId, msgType, requestId)
		return nil
	}

	var responseToSend string
	var marshalingError error

	switch msgType {
	case "EVALUATION_RESULT":
		req.Response = msgMap["result"]
		if isException, excOK := msgMap["isException"].(bool); excOK && isException {
			req.err = fmt.Errorf("script evaluation exception: %v", msgMap["exceptionInfo"])
			req.Response = msgMap["exceptionInfo"] // Overwrite Response with exceptionInfo
		}
		if req.err != nil {
			responseToSend = fmt.Sprintf("Error: %v", req.err)
		} else {
			responseBytes, err := json.Marshal(req.Response)
			if err != nil {
				responseToSend = fmt.Sprintf("Error marshalling eval response result: %v", err)
				marshalingError = fmt.Errorf("error marshalling eval response: %w", err)
			} else {
				responseToSend = string(responseBytes)
			}
		}
		log.Printf("Client %s: Processed EVALUATION_RESULT for %s.", t.ClientId, requestId)

	case "ELEMENTS_SCREENSHOT_RESULT":
		req.Response = msgMap["imageData"] // This should be a map or the error string
		if errMsg, errOK := msgMap["error"].(string); errOK && errMsg != "" {
			req.err = fmt.Errorf("screenshot error from client: %s", errMsg)
			// If there's a top-level error, Response might just be that error string
			errorResponsePayload := map[string]string{"error": errMsg}
			responseBytes, err := json.Marshal(errorResponsePayload)
			if err != nil {
				responseToSend = fmt.Sprintf(`{"error": "Error marshalling screenshot error: %v"}`, err)
				marshalingError = fmt.Errorf("error marshalling screenshot error response: %w", err)
			} else {
				responseToSend = string(responseBytes)
			}
		} else {
			// If no top-level error, marshal the imageData map
			responseBytes, err := json.Marshal(req.Response)
			if err != nil {
				responseToSend = fmt.Sprintf(`{"error": "Error marshalling screenshot imageData: %v"}`, err)
				marshalingError = fmt.Errorf("error marshalling screenshot imageData: %w", err)
			} else {
				responseToSend = string(responseBytes)
			}
		}
		log.Printf("Client %s: Processed ELEMENTS_SCREENSHOT_RESULT for %s.", t.ClientId, requestId)

	default:
		log.Printf("Client %s: Received unhandled message type '%s' from client for requestId %s.", t.ClientId, msgType, requestId)
		return nil // Don't mark as finished or send to channel if unhandled
	}

	req.ResponseAt = time.Now()
	if marshalingError != nil && req.err == nil { // If marshaling failed, set it as the primary error
		req.err = marshalingError
		if responseToSend == "" { // Ensure responseToSend has error info
			responseToSend = fmt.Sprintf("Error: %v", req.err)
		}
	}
	req.finished = true

	// Send to channel if it's still open (it should be for a waiting request)
	// A select with a default might be safer if the channel could somehow be closed by a timeout race,
	// but `close(req.recvChan)` happens after this send or on timeout.
	select {
	case req.recvChan <- responseToSend:
		log.Printf("Client %s: Sent response for requestId %s to recvChan.", t.ClientId, requestId)
	default:
		// This case might occur if the channel was closed due to a timeout just before this send attempt.
		log.Printf("Client %s: recvChan for requestId %s was already closed. Response not sent.", t.ClientId, requestId)
	}

	close(req.recvChan) // Close channel after sending or if it was already closed
	delete(t.handler.pendingRequests, requestId)

	return nil
}

func (t *Conn) OnTimeout() bool {
	log.Printf("Client %s: WebSocket connection timeout.", t.ClientId)
	return true
}

func (c *Conn) OnClose() {
	writer := c.JSONConn.Writer
	if writer != nil && writer.SendChan() != nil {
		c.handler.withClientFanout(c.ClientId, false, func(fanout *conc.FanOut[conc.Message[any]]) {
			if fanout != nil {
				removed := fanout.Remove(writer.SendChan(), true)
				<-removed
				log.Printf("Client %s: Writer removed from fanout. Remaining in fanout: %d", c.ClientId, fanout.Count())
			}
		})
	} else {
		log.Printf("Client %s: OnClose called, but writer or SendChan was nil.", c.ClientId)
	}
	log.Printf("Client %s: WebSocket connection closed.", c.ClientId)
}

func (c *Conn) OnStart(conn *websocket.Conn) error {
	if err := c.JSONConn.OnStart(conn); err != nil {
		log.Printf("Client %s: Error in embedded JSONConn.OnStart: %v", c.ClientId, err)
		return err
	}

	writer := c.JSONConn.Writer
	if writer == nil || writer.SendChan() == nil {
		log.Printf("Client %s: JSONConn.Writer or SendChan not initialized in OnStart. Cannot add to fanout.", c.ClientId)
		return fmt.Errorf("JSONConn writer not initialized for client %s", c.ClientId)
	}

	log.Printf("Client %s: New WebSocket connection started. Adding to fanout.", c.ClientId)

	c.handler.withClientFanout(c.ClientId, true, func(fanout *conc.FanOut[conc.Message[any]]) {
		fanout.Add(writer.SendChan(), nil, false)
		log.Printf("Client %s: Registered new connection. Total for this agent in fanout: %d", c.ClientId, fanout.Count())
	})

	go func() {
		time.Sleep(1 * time.Second)
		welcomeScript := "console.log('[AgentWelcome] Go backend says hello! Location: ' + window.location.href + '. Title: ' + document.title + '. Timestamp: ' + new Date().toLocaleTimeString()); ({ pageTitle: document.title, userAgent: navigator.userAgent, connectionTime: new Date().toISOString() });"
		req := NewRequest(c.ClientId, welcomeScript) // This will create a new request, okay for welcome.
		c.handler.SubmitRequest("EVALUATE_SCRIPT", req)
		log.Printf("Client %s: Sent welcome EVALUATE_SCRIPT.", c.ClientId)
	}()
	return nil
}

func (h *Handler) SubmitRequest(reqType string, req *Request) {
	h.reqMutex.Lock()
	h.pendingRequests[req.Id] = req
	h.reqMutex.Unlock()
	log.Printf("Stored pending eval request %s for client %s. Total pending: %d", req.Id, req.ClientId, len(h.pendingRequests))

	messagePayload := map[string]any{
		"type":      reqType,
		"requestId": req.Id,
		"payload":   req.Payload,
	}
	h.BroadcastToAgent(req.ClientId, messagePayload)
}

func (h *Handler) WaitForRequest(req *Request) (response string, ok bool, timedout bool) {
	clientId := req.ClientId
	log.Printf("Client %s /eval: Waiting for response on recvChan for ReqID: %s", clientId, req.Id)
	select {
	case responseMsg, ok := <-req.recvChan:
		if ok {
			log.Printf("Client %s /eval: Received response for ReqID %s: %s", clientId, req.Id, responseMsg)
		} else {
			log.Printf("Client %s /eval: recvChan closed without a message for ReqID %s.", clientId, req.Id)
		}
		return responseMsg, ok, false
	case <-time.After(req.Timeout): // Standard timeout for eval
		log.Printf("Client %s /eval: Timed out waiting for response for ReqID %s", clientId, req.Id)
		h.reqMutex.Lock()
		if pendingReq, exists := h.pendingRequests[req.Id]; exists && !pendingReq.finished {
			pendingReq.finished = true // Mark as finished to prevent further processing
			close(pendingReq.recvChan) // Close chan to unblock any potential race
			delete(h.pendingRequests, req.Id)
			log.Printf("Client %s /eval: Cleaned up timed-out pending request %s", clientId, req.Id)
		}
		h.reqMutex.Unlock()
		return "", false, true
	}
}

func (h *Handler) BroadcastToAgent(clientId string, payload map[string]any) {
	h.withClientFanout(clientId, false, func(fanout *conc.FanOut[conc.Message[any]]) {
		if fanout != nil {
			log.Printf("Broadcasting to fanout for client %s (count %d): %v", clientId, fanout.Count(), payload)
			fanout.Send(conc.Message[any]{Value: payload})
		} else {
			log.Printf("Cannot broadcast: No fanout found for client %s (and ensure=false). Payload: %v", clientId, payload)
		}
	})
}

func (h *Handler) withClientFanout(clientId string, ensure bool, callback func(fanout *conc.FanOut[conc.Message[any]])) {
	h.fanoutMutex.Lock()
	fanout, exists := h.Fanouts[clientId]
	if !exists && ensure {
		fanout = conc.NewFanOut[conc.Message[any]](nil)
		h.Fanouts[clientId] = fanout
		log.Printf("Created new fanout for ClientId: %s", clientId)
	}
	h.fanoutMutex.Unlock()

	if fanout != nil {
		callback(fanout)
	} else if !ensure {
		log.Printf("Fanout for client %s is nil and ensure=false, callback not executed.", clientId)
	}
}

func NewServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	handler := NewHandler()

	mux.HandleFunc("GET /agents/{clientId}/subscribe", gohttp.WSServe(handler, nil))

	mux.HandleFunc("POST /agents/{clientId}/eval", func(w http.ResponseWriter, r *http.Request) {
		clientId := r.PathValue("clientId")
		if clientId == "" {
			http.Error(w, "Client ID is missing in path", http.StatusBadRequest)
			return
		}
		wait := r.URL.Query().Get("wait") == "true"

		scriptBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Client %s /eval: Could not read script body: %v", clientId, err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		script := string(scriptBytes)
		if script == "" {
			http.Error(w, "Script body cannot be empty", http.StatusBadRequest)
			return
		}

		req := NewRequest(clientId, script)
		handler.SubmitRequest("EVALUATE_SCRIPT", req)

		w.Header().Set("Content-Type", "application/json")
		if wait {
			response, ok, timedout := handler.WaitForRequest(req)
			if timedout {
				http.Error(w, "Timeout waiting for script evaluation response", http.StatusGatewayTimeout)
			} else if ok {
				w.Header().Set("Content-Type", "application/json")
				// response is expected to be a JSON string of the result or an error string
				// If it's an error string from our side, it might not be valid JSON, so wrap it.
				// If it's from the client, it should be marshal-able JSON or a simple string.
				// For simplicity, we assume if it starts with "Error:", it's not JSON from client.
				if !json.Valid([]byte(response)) {
					// Attempt to re-marshal if it's likely an error string we created
					// This part could be more robust based on how errors are formatted
					errorPayload := map[string]any{"error": response}
					jsonError, _ := json.Marshal(errorPayload)
					fmt.Fprintf(w, `{"requestId": "%s", "response": %s}`, req.Id, string(jsonError))
				} else {
					fmt.Fprintf(w, `{"requestId": "%s", "response": %s}`, req.Id, response)
				}
			} else {
				http.Error(w, "Response channel closed or request processed/timed out.", http.StatusInternalServerError)
			}
		} else {
			json.NewEncoder(w).Encode(map[string]string{
				"status":    "EVALUATE_SCRIPT command sent",
				"requestId": req.Id,
			})
		}
	})

	mux.HandleFunc("POST /agents/{clientId}/screenshots", func(w http.ResponseWriter, r *http.Request) {
		clientId := r.PathValue("clientId")
		if clientId == "" {
			http.Error(w, "Client ID is missing in path", http.StatusBadRequest)
			return
		}
		wait := r.URL.Query().Get("wait") == "true"

		var requestBody struct {
			Selectors []string `json:"selectors"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if len(requestBody.Selectors) == 0 {
			http.Error(w, "Selectors array cannot be empty", http.StatusBadRequest)
			return
		}

		req := NewRequest(clientId, requestBody.Selectors)
		handler.SubmitRequest("CAPTURE_ELEMENTS_SCREENSHOT", req)

		if wait {
			response, ok, timedout := handler.WaitForRequest(req)
			if timedout {
				http.Error(w, "Timeout waiting for script evaluation response", http.StatusGatewayTimeout)
			} else if ok {
				// responseMsg is expected to be a JSON string of the imageData map or an error structure
				fmt.Fprintf(w, `{"requestId": "%s", "response": %s}`, req.Id, response)
			} else {
				http.Error(w, "Response channel closed or request processed/timed out.", http.StatusInternalServerError)
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "CAPTURE_ELEMENTS_SCREENSHOT command sent", "requestId": req.Id})
		}
	})

	mux.HandleFunc("GET /test_eval", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.URL.Query().Get("agent")
		script := r.URL.Query().Get("script")
		if agentName == "" || script == "" {
			http.Error(w, "Missing 'agent' or 'script' query parameter", http.StatusBadRequest)
			return
		}
		req := NewRequest(agentName, script)
		handler.SubmitRequest("EVALUATE_SCRIPT", req)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Sent EVALUATE_SCRIPT to agent %s (ReqID: %s). Script: '%s'", agentName, req.Id, script)
	})

	log.Println("WebSocket ServeMux configured.")
	log.Println("GET  /agents/{clientId}/subscribe 		- WebSocket connections")
	log.Println("POST /agents/{clientId}/eval        	- Evaluate script (add ?wait=true to wait for response)")
	log.Println("POST /agents/{clientId}/screenshots	- Capture screenshots of elements (add ?wait=true to wait for response)")
	log.Println("GET  /test_eval?agent=<name>&script=<javascript> - Test script evaluation")
	return mux
}
