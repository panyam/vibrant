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
	Response   any // Stores the 'result' field from EVALUATION_RESULT
	ResponseAt time.Time
	err        error       // Stores error if evaluation resulted in an exception
	finished   bool        // To prevent processing a response multiple times
	recvChan   chan string // For the /eval?wait=true handler, made buffered
}

func NewRequest(clientId string, payload any) *Request {
	return &Request{
		Id:       fmt.Sprintf("eval-%d-%s", time.Now().UnixNano(), clientId),
		SentAt:   time.Now(),
		ClientId: clientId,
		Payload:  payload,
		recvChan: make(chan string, 1), // Buffered channel of size 1
	}
}

// Json method of Request is not directly used for WebSocket message, but good for logging/internal state
func (r *Request) Json() any {
	return map[string]any{
		"Payload":   r.Payload, // This is the script for eval requests
		"RequestId": r.Id,
	}
}

type Handler struct {
	fanoutMutex     sync.RWMutex
	Fanouts         map[string]*conc.FanOut[conc.Message[any]]
	reqMutex        sync.RWMutex
	pendingRequests map[string]*Request // Stores requests waiting for EVALUATION_RESULT
}

func NewHandler() *Handler {
	return &Handler{
		Fanouts:         make(map[string]*conc.FanOut[conc.Message[any]]),
		pendingRequests: make(map[string]*Request),
	}
}

type Conn struct {
	gohttp.JSONConn // Embeds the connection helper from goutils
	handler         *Handler
	ClientId        string
}

// Validate is called by gohttp.WSServe to validate the connection request
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

// HandleMessage is called by gohttp.JSONConn when a message is received from the WebSocket client
func (t *Conn) HandleMessage(msgData any) error {
	msgMap, ok := msgData.(map[string]interface{})
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

	if msgType == "EVALUATION_RESULT" {
		t.handler.reqMutex.Lock()
		req, exists := t.handler.pendingRequests[requestId]
		if exists {
			if !req.finished { // Process only if not already finished
				req.Response = msgMap["result"]
				req.ResponseAt = time.Now()
				if isException, excOK := msgMap["isException"].(bool); excOK && isException {
					req.err = fmt.Errorf("script evaluation exception: %v", msgMap["exceptionInfo"])
					req.Response = msgMap["exceptionInfo"] // Overwrite Response with exceptionInfo
				}
				req.finished = true // Mark finished before sending to channel

				var responseString string
				if req.err != nil {
					responseString = fmt.Sprintf("Error: %v", req.err)
				} else {
					responseBytes, err := json.Marshal(req.Response)
					if err != nil {
						responseString = fmt.Sprintf("Error marshalling response result: %v", err)
						req.err = fmt.Errorf("error marshalling response: %w", err)
					} else {
						responseString = string(responseBytes)
					}
				}

				// recvChan is now buffered, direct send is fine.
				req.recvChan <- responseString
				log.Printf("Client %s: Sent response for requestId %s to buffered recvChan.", t.ClientId, requestId)
				close(req.recvChan)

				delete(t.handler.pendingRequests, requestId)

			} else {
				log.Printf("Client %s: Received duplicate/late EVALUATION_RESULT for already finished requestId %s.", t.ClientId, requestId)
			}
		} else {
			log.Printf("Client %s: Received EVALUATION_RESULT for unknown or already processed/deleted requestId %s.", t.ClientId, requestId)
		}
		t.handler.reqMutex.Unlock()
	} else {
		log.Printf("Client %s: Received unhandled message type '%s' from client.", t.ClientId, msgType)
	}
	return nil
}

func (t *Conn) OnTimeout() bool {
	log.Printf("Client %s: WebSocket connection timeout.", t.ClientId)
	return true // Returning true usually means close the connection
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

		_ = c.handler.SubmitEvalRequest(c.ClientId, welcomeScript)
		log.Printf("Client %s: Sent welcome EVALUATE_SCRIPT.", c.ClientId)
	}()
	return nil
}

func (h *Handler) SubmitEvalRequest(clientId string, script string) *Request {
	req := NewRequest(clientId, script)

	h.reqMutex.Lock()
	h.pendingRequests[req.Id] = req
	h.reqMutex.Unlock()
	log.Printf("Stored pending request %s for client %s. Total pending: %d", req.Id, clientId, len(h.pendingRequests))

	messagePayload := map[string]any{
		"type":             "EVALUATE_SCRIPT",
		"requestId":        req.Id,
		"scriptToEvaluate": script,
	}
	h.BroadcastToAgent(clientId, messagePayload)
	return req
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

		req := handler.SubmitEvalRequest(clientId, script)

		if wait {
			log.Printf("Client %s /eval: Waiting for response on recvChan for ReqID: %s", clientId, req.Id)
			select {
			case responseMsg, ok := <-req.recvChan:
				if ok {
					log.Printf("Client %s /eval: Received response for ReqID %s: %s", clientId, req.Id, responseMsg)
					w.Header().Set("Content-Type", "application/json")
					// Ensure responseMsg is valid for direct inclusion or re-marshal if it's complex object string
					// For simplicity, assuming responseMsg is either simple string, number, or already a JSON string of an object/array.
					fmt.Fprintf(w, `{"requestId": "%s", "response": %s}`, req.Id, responseMsg)
				} else {
					log.Printf("Client %s /eval: recvChan closed without a message for ReqID %s.", clientId, req.Id)
					http.Error(w, "Response channel closed or request processed/timed out.", http.StatusInternalServerError)
				}
			case <-time.After(30 * time.Second):
				log.Printf("Client %s /eval: Timed out waiting for response for ReqID %s", clientId, req.Id)
				http.Error(w, "Timeout waiting for script evaluation response", http.StatusGatewayTimeout)
				handler.reqMutex.Lock()
				if pendingReq, exists := handler.pendingRequests[req.Id]; exists && !pendingReq.finished {
					pendingReq.finished = true
					close(pendingReq.recvChan)
					delete(handler.pendingRequests, req.Id)
					log.Printf("Client %s /eval: Cleaned up timed-out pending request %s", clientId, req.Id)
				}
				handler.reqMutex.Unlock()
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":    "EVALUATE_SCRIPT command sent",
				"requestId": req.Id,
			})
		}
	})

	mux.HandleFunc("GET /test_eval", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.URL.Query().Get("agent")
		script := r.URL.Query().Get("script")
		if agentName == "" || script == "" {
			http.Error(w, "Missing 'agent' or 'script' query parameter", http.StatusBadRequest)
			return
		}
		req := handler.SubmitEvalRequest(agentName, script)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Sent EVALUATE_SCRIPT to agent %s (ReqID: %s). Script: '%s'", agentName, req.Id, script)
	})

	log.Println("WebSocket ServeMux configured.")
	log.Println("GET  /agents/{clientId}/subscribe - WebSocket connections")
	log.Println("POST /agents/{clientId}/eval        - Evaluate script (add ?wait=true to wait for response)")
	log.Println("GET  /test_eval?agent=<name>&script=<javascript> - Test script evaluation")
	return mux
}
