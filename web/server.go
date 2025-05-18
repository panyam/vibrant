package web

import (
	"log"
	"net/http"
	"strings" // Added for path manipulation
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default for local development.
		// For production, you should implement proper origin checking.
		return true
	},
}

// connectionManager holds active WebSocket connections.
type connectionManager struct {
	sync.RWMutex
	connections map[string]map[*websocket.Conn]bool
}

// NewConnectionManager creates a new connection manager.
func NewConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[string]map[*websocket.Conn]bool),
	}
}

// AddConnection registers a new WebSocket connection.
func (cm *connectionManager) AddConnection(connectionName string, conn *websocket.Conn) {
	cm.Lock()
	defer cm.Unlock()
	if _, ok := cm.connections[connectionName]; !ok {
		cm.connections[connectionName] = make(map[*websocket.Conn]bool)
	}
	cm.connections[connectionName][conn] = true
	log.Printf("Registered new connection for agent: %s. Total for this agent: %d", connectionName, len(cm.connections[connectionName]))
}

// RemoveConnection unregisters a WebSocket connection.
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

// BroadcastToAgent sends a message to all clients connected under a specific agent name.
func (cm *connectionManager) BroadcastToAgent(connectionName string, messageType int, message []byte) {
	cm.RLock()
	defer cm.RUnlock()

	subscribers, ok := cm.connections[connectionName]
	if !ok {
		// log.Printf("No subscribers found for agent: %s", connectionName)
		return
	}

	log.Printf("Broadcasting message to %d subscribers for agent: %s", len(subscribers), connectionName)
	for conn := range subscribers {
		err := conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("Error writing message to subscriber for agent %s: %v. Removing connection.", connectionName, err)
			// Clean up the bad connection. Need to acquire write lock.
			// Schedule removal outside the read lock or use a more complex cleanup mechanism.
			// For simplicity here, we'll log and the read loop on the other side will eventually detect closure.
		}
	}
}

// Global connection manager instance
var manager = NewConnectionManager()

// serveWs handles WebSocket requests from the peer.
// The path is expected to be /agent/{connectionName}/subscribe from the perspective of this mux
func serveWs(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Expected path: "agent/{connectionName}/subscribe"
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
		// Continue even if welcome message fails
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

// NewServeMux creates and configures an http.ServeMux with the WebSocket handler.
// It expects to be mounted under a prefix like "/agents", so paths here are relative to that.
func NewServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	// This handler will match /agent/{connectionName}/subscribe
	// after http.StripPrefix("/agents", agentsMux) in main.go
	mux.HandleFunc("/agent/", serveWs) // serveWs will further parse the path

	// Test broadcast endpoint, relative to the mount point of this mux.
	// So, if mounted at /agents, this becomes /agents/test_broadcast
	mux.HandleFunc("/test_broadcast", func(w http.ResponseWriter, r *http.Request) {
		agentName := r.URL.Query().Get("agent")
		message := r.URL.Query().Get("msg")
		if agentName == "" || message == "" {
			http.Error(w, "Missing 'agent' or 'msg' query parameter", http.StatusBadRequest)
			return
		}
		log.Printf("Test broadcast triggered for agent '%s' with message '%s'", agentName, message)
		
		jsonData := `{"event": "test", "payload": "` + message + `"}` // Basic JSON construction
		manager.BroadcastToAgent(agentName, websocket.TextMessage, []byte(jsonData))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Broadcast initiated for agent: " + agentName))
	})

	log.Println("Websocket ServeMux configured. Mount under a prefix (e.g., /agents).")
	log.Println("WebSocket handler expects paths like: /agent/{connectionName}/subscribe (relative to mount point)")
	log.Println("Test broadcast endpoint expects: /test_broadcast?agent=<name>&msg=<message> (relative to mount point)")
	return mux
}

// GetConnectionManager returns the global connection manager instance.
// This allows other parts of your application (outside this package)
// to access the manager and broadcast messages if needed, though it's often
// better to expose a dedicated broadcasting function or channel.
func GetConnectionManager() *connectionManager {
    return manager
}
