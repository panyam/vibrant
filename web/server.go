package web

import (
	"encoding/json" // For parsing request body and constructing messages
	"io"            // For reading request body
	"log"
	"net/http"
	"strings"
	"sync"
	"time" // For requestId generation (simple example)

	"github.com/gorilla/websocket"
)

// upgrader (no change)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default for local development.
		// For production, you should implement proper origin checking.
		return true
	},
}

// connectionManager and its methods (no change)
type connectionManager struct {
	sync.RWMutex
	connections map[string]map[*websocket.Conn]bool
}

func NewConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[string]map[*websocket.Conn]bool),
	}
}

func (cm *connectionManager) AddConnection(connectionName string, conn *websocket.Conn) {
	cm.Lock()
	defer cm.Unlock()
	if _, ok := cm.connections[connectionName]; !ok {
		cm.connections[connectionName] = make(map[*websocket.Conn]bool)
	}
	cm.connections[connectionName][conn] = true
	log.Printf("Registered new connection for agent: %s. Total for this agent: %d", connectionName, len(cm.connections[connectionName]))
}

func (cm *connectionManager) RemoveConnection(connectionName string, conn *websocket.Conn) {
	cm.Lock()
	defer cm.Unlock()
	if conns, ok := cm.connections[connectionName]; ok {
		if _, ok := conns[conn]; ok {
			delete(cm.connections[connectionName], conn)
			log.Printf("Unregistered connection for agent: %s. Remaining for this agent: %d", connectionName, len(cm.connections[connectionName]))
			if len(cm.connections[connectionName]) == 0 {
				delete(cm.connections, connectionName)
				log.Printf("No more connections for agent: %s. Removed agent entry.", connectionName)
			}
		}
	}
}

func (cm *connectionManager) BroadcastToAgent(connectionName string, messageType int, message []byte) {
	cm.RLock()
	defer cm.RUnlock()
	subscribers, ok := cm.connections[connectionName]
	if !ok {
		log.Printf("BroadcastToAgent: No subscribers found for agent: %s", connectionName)
		return
	}
	log.Printf("Broadcasting message to %d subscribers for agent: %s, Message: %s", len(subscribers), connectionName, string(message))
	for conn := range subscribers {
		err := conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("Error writing message to subscriber for agent %s: %v.", connectionName, err)
		}
	}
}

var manager = NewConnectionManager()

// serveWs (no change from your provided version)
func serveWs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) != 3 || parts[0] != "agent" || parts[2] != "subscribe" {
		log.Printf("Invalid WebSocket path structure for serveWs: %s. Expected /agent/{connectionName}/subscribe", path)
		http.Error(w, "Invalid WebSocket path structure.", http.StatusBadRequest)
		return
	}
	connectionName := parts[1]
	log.Printf("Incoming WebSocket connection attempt for agent: %s from %s (path: %s)", connectionName, r.RemoteAddr, r.URL.Path)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection for agent %s: %v", connectionName, err)
		return
	}
	defer conn.Close()
	manager.AddConnection(connectionName, conn)
	defer manager.RemoveConnection(connectionName, conn)
	welcomeMsg := []byte(`{"type": "status", "message": "Successfully connected to agent ` + connectionName + `"}`)
	if err := conn.WriteMessage(websocket.TextMessage, welcomeMsg); err != nil {
		log.Printf("Failed to send welcome message to agent %s: %v", connectionName, err)
	}
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived, websocket.CloseNormalClosure) {
				log.Printf("Client %s for agent %s disconnected with error: %v", conn.RemoteAddr().String(), connectionName, err)
			} else if e, ok := err.(*websocket.CloseError); ok && (e.Code == websocket.CloseNormalClosure || e.Code == websocket.CloseGoingAway) {
				log.Printf("Client %s for agent %s disconnected gracefully (code %d).", conn.RemoteAddr().String(), connectionName, e.Code)
			} else {
				log.Printf("Error reading message from client %s for agent %s: %v", conn.RemoteAddr().String(), connectionName, err)
			}
			break
		}
		log.Printf("Received message from agent %s (type %d): %s (Note: Server doesn't process these)", connectionName, messageType, string(p))
	}
}

// handleAgentCommandPost handles POST requests to /agents/<connectionName>/<commandType>
func handleAgentCommandPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// Expected path structure: "agents/{connectionName}/{commandType}"
	if len(parts) != 3 || parts[0] != "agents" {
		log.Printf("Invalid command path structure: %s. Expected /agents/{connectionName}/{commandType}", r.URL.Path)
		http.Error(w, "Invalid command path structure", http.StatusBadRequest)
		return
	}

	connectionName := parts[1]
	commandType := parts[2] // This is the actual command type like SCROLL_TO_TOP, QUERY_SELECTOR_ALL etc.

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body for command %s on agent %s: %v", commandType, connectionName, err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Unmarshal the body to a map to inject/override the 'type' field
	var commandPayload map[string]interface{}
	if len(bodyBytes) > 0 {
		err = json.Unmarshal(bodyBytes, &commandPayload)
		if err != nil {
			// If body is not valid JSON, but commandType doesn't require a body (e.g. SCROLL_TO_TOP)
			// we might proceed, or decide to be strict. For now, let's be strict if body exists.
			log.Printf("Error unmarshalling JSON body for command %s on agent %s: %v. Body: %s", commandType, connectionName, err, string(bodyBytes))
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		commandPayload = make(map[string]interface{})
	}

	// Set/Override the type from the URL path parameter
	// Convert commandType from path (e.g., "scrollToTop") to the message type ("SCROLL_TO_TOP")
	// For simplicity, we'll assume commandType in URL is already in the correct case (e.g. SCROLL_TO_TOP)
	// In a real app, you might want to map camelCase path to UPPER_SNAKE_CASE type or vice-versa.
	commandPayload["type"] = commandType

	// Add a requestId for commands that might expect a response or need tracking (like querySelectorAll)
	if commandType == "QUERY_SELECTOR_ALL" {
		if _, ok := commandPayload["requestId"]; !ok {
			commandPayload["requestId"] = time.Now().UnixNano() // Simple unique ID
		}
	}

	finalMessageBytes, err := json.Marshal(commandPayload)
	if err != nil {
		log.Printf("Error marshalling final command message for %s on agent %s: %v", commandType, connectionName, err)
		http.Error(w, "Error preparing message", http.StatusInternalServerError)
		return
	}

	log.Printf("Received command via POST: Agent=%s, CommandType=%s, Body=%s", connectionName, commandType, string(bodyBytes))
	manager.BroadcastToAgent(connectionName, websocket.TextMessage, finalMessageBytes)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "success",
		"agent":       connectionName,
		"commandType": commandType,
		"message":     "Command broadcasted",
	})
}

func NewServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Handles WebSocket connections: /agent/{connectionName}/subscribe
	mux.HandleFunc("/agent/", serveWs) // serveWs parses the full path like /agent/name/subscribe

	// Handles command POSTs: /agents/{connectionName}/{commandType}
	mux.HandleFunc("/agents/", handleAgentCommandPost) // handleAgentCommandPost parses this

	// Test broadcast endpoint (kept for now)
	mux.HandleFunc("/test_broadcast", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.URL.Query().Get("agent")
		message := r.URL.Query().Get("msg")
		commandType := r.URL.Query().Get("type") // Optional type for GET test

		if agentName == "" || message == "" {
			http.Error(w, "Missing 'agent' or 'msg' query parameter", http.StatusBadRequest)
			return
		}
		if commandType == "" {
			commandType = "test" // Default if not specified
		}

		log.Printf("Test broadcast (GET) triggered for agent '%s', type '%s', message '%s'", agentName, commandType, message)

		// Construct a JSON message - this matches what the Chrome extension expects
		payloadMap := map[string]interface{}{"payload": message}
		if commandType == "SCROLL_DELTA" { // Example for specific GET test
			payloadMap["deltaY"] = 100 // Default test delta
		}

		jsonDataMap := map[string]interface{}{
			"type": commandType,
		}
		// Merge payloadMap into jsonDataMap
		for k, v := range payloadMap {
			jsonDataMap[k] = v
		}

		jsonDataBytes, _ := json.Marshal(jsonDataMap)
		manager.BroadcastToAgent(agentName, websocket.TextMessage, jsonDataBytes)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET Test Broadcast initiated for agent: " + agentName + " with type " + commandType))
	})

	log.Println("WebSocket ServeMux configured.")
	log.Println("POST commands to: /agents/{connectionName}/{COMMAND_TYPE} (e.g. /agents/myClient/SCROLL_TO_TOP)")
	log.Println("WebSocket connections at: /agent/{connectionName}/subscribe")
	return mux
}

func GetConnectionManager() *connectionManager {
	return manager
}
