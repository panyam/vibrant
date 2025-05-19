package web

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gohttp "github.com/panyam/goutils/http"
)

// An asynch request sent to the plugin where response is handled asynchronously
type Request struct {
	Id         string
	ClientId   string
	Payload    any
	SentAt     time.Time
	Response   any
	ResponseAt time.Time
	err        error
	finished   bool
	recvChan   chan string
}

func NewRequest(clientId string, payload any) *Request {
	return &Request{
		Id:       fmt.Sprintf("eval-%d-%s", time.Now().UnixNano(), clientId),
		SentAt:   time.Now(),
		ClientId: clientId,
		Payload:  payload,
		recvChan: make(chan string),
	}
}

func (r *Request) Json() any {
	return map[string]any{
		"Payload":   r.Payload,
		"RequestId": r.Id,
	}
}

type Handler struct {
	// Mapping connection Ids -> list of fanouts.
	// A fanout ensures that N connections can be broadcasted easily
	fanoutMutex sync.RWMutex
	Fanouts     map[string]*conc.FanOut[conc.Message[any]]

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
	// get the connection id from the request
	out = &Conn{handler: h}
	isValid = true
	out.ClientId = r.PathValue("clientId")
	log.Println("Are we here???")
	return
}

type connectionManager struct {
	connections map[string]map[*websocket.Conn]bool
}

func (t *Conn) HandleMessage(msg any) error {
	log.Println("Received Message: ", msg)
	// get the request and call its channel if not already handled
	return nil
}

func (t *Conn) OnTimeout() bool {
	return false
}

func (c *Conn) OnClose() {
	writer := c.JSONConn.Writer

	// Removal can be synchronous or asynchronous - we want to ensure it is done
	// synchronously so another publish (if one came in) wont be attempted on a closed channel
	c.handler.withClientFanout(c.ClientId, true, func(fanout *conc.FanOut[conc.Message[any]]) {
		<-fanout.Remove(writer.SendChan(), true)
		log.Printf("Removed agent entry: %s", c.ClientId)
	})
	c.JSONConn.OnClose()
}

// Called when a new connection is setup
func (c *Conn) OnStart(conn *websocket.Conn) error {
	c.JSONConn.OnStart(conn)
	writer := c.JSONConn.Writer

	log.Println("Got a new connection.....")

	c.handler.withClientFanout(c.ClientId, true, func(fanout *conc.FanOut[conc.Message[any]]) {
		// Register the writer channel into the fanout
		fanout.Add(writer.SendChan(), nil, false)
		log.Printf("Registered new connection for agent: %s. Total for this agent: %d", c.ClientId, fanout.Count())
	})

	go func() {
		time.Sleep(1 * time.Second)
		scriptPayload := map[string]interface{}{
			"type":             "EVALUATE_SCRIPT",
			"requestId":        fmt.Sprintf("welcome-%d", time.Now().UnixNano()),
			"scriptToEvaluate": "console.log('[AgentWelcome] Go backend says hello! Location: ' + window.location.href + '. Title: ' + document.title); ({ pageTitle: document.title, userAgent: navigator.userAgent });",
		}
		c.handler.BroadcastToAgent(c.ClientId, scriptPayload)
	}()

	// Now send an eval after 1 second on this connectio
	return nil
}

func (h *Handler) SubmitEvalRequest(clientId string, script string) *Request {
	h.reqMutex.Lock()
	defer h.reqMutex.Unlock()

	req := NewRequest(clientId, map[string]any{"scriptToEvaluate": script})

	message := map[string]any{
		"type":      "EVALUATE_SCRIPT",
		"requestId": req.Id,
		"script":    script,
	}
	h.BroadcastToAgent(clientId, message)
	return req
}

func (h *Handler) BroadcastToAgent(clientId string, payload map[string]any) {
	h.withClientFanout(clientId, false, func(fanout *conc.FanOut[conc.Message[any]]) {
		fanout.Send(conc.Message[any]{Value: payload})
	})
}

func (h *Handler) withClientFanout(clientId string, ensure bool, callback func(fanout *conc.FanOut[conc.Message[any]])) {
	h.fanoutMutex.RLock()
	defer h.fanoutMutex.RUnlock()
	fanout := h.Fanouts[clientId]
	if fanout == nil {
		if ensure {
			fanout = conc.NewFanOut[conc.Message[any]](nil)
			h.Fanouts[clientId] = fanout
		} else {
			log.Println("No fan out found for client: ", clientId)
		}
	}
	callback(fanout)
}

func NewServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	// A few simple handlers
	handler := NewHandler()

	// Plugin subscribes to websocket via this endpoint
	mux.HandleFunc("/agents/{clientId}/subscribe", gohttp.WSServe(handler, nil))

	// POST request with the body containing the JS to be evaluated by the plugin on the page
	// This will also send a requestId that is genreated as part of the payload so the plugin
	// can respond back Sends eval scriptCommands to be evaluated can be published using this endpoint
	// Behind the scenes a request Id is generated and sent on the WS
	// the EVALUATE command is used to send snippets of JS to evaluate on the host page
	mux.HandleFunc("/agents/{clientId}/eval/", func(w http.ResponseWriter, r *http.Request) {
		clientId := r.PathValue("clientId")
		wait := r.URL.Query().Get("wait") == "true" // whether to wait for a response

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed for /eval endpoint", http.StatusMethodNotAllowed)
			return
		}

		script, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Could not read script body: ", err)
		}
		defer r.Body.Close()

		req := handler.SubmitEvalRequest(clientId, string(script))

		if wait {
			msg := <-req.recvChan
			log.Println("Received Response: ", msg)
			fmt.Fprintf(w, "Published Message Successfully, RequestId: %s, Response: \n%s", req.Id, msg)
		} else {
			fmt.Fprintf(w, "Published Message Successfully, RequestId: %s", req.Id)
		}
	})

	log.Println("WebSocket ServeMux configured.")
	log.Println("WebSocket connections at: /agents/{connectionName}/subscribe")
	log.Println("POST script evaluations to: /agents/{connectionName}/eval")
	log.Println("Test script evaluation via GET: /test_eval?agent=<name>&script=<javascript>")
	return mux
}
