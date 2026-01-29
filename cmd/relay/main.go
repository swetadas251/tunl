package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// ============================================
// DATA STRUCTURES
// ============================================

// Tunnel represents one connected client
type Tunnel struct {
	ID       string
	Conn     *websocket.Conn
	Requests map[string]chan *ResponsePayload // waiting for responses
	ReqMu    sync.Mutex
}

// Message is the envelope for all communication
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// RegisteredPayload tells the client their public URL
type RegisteredPayload struct {
	URL       string `json:"url"`
	Subdomain string `json:"subdomain"`
}

// RequestPayload is an HTTP request to forward
type RequestPayload struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// ResponsePayload is the HTTP response coming back
type ResponsePayload struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// ============================================
// GLOBAL STATE
// ============================================

// Store all active tunnels: subdomain -> Tunnel
var tunnels = make(map[string]*Tunnel)
var tunnelsMu sync.RWMutex

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// ============================================
// MAIN
// ============================================

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Route: WebSocket connections from tunnel clients
	http.HandleFunc("/tunnel", handleTunnelConnection)

	// Route: All other requests go to tunnels
	http.HandleFunc("/", handlePublicRequest)

	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("  🌐 tunl relay server")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("  HTTP Server:  http://localhost:%s\n", port)
	fmt.Printf("  WebSocket:    ws://localhost:%s/tunnel\n", port)
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("  Waiting for tunnel clients to connect...")
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ============================================
// WEBSOCKET HANDLER (for tunnel clients)
// ============================================

func handleTunnelConnection(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	// Generate unique subdomain for this tunnel
	subdomain := generateID()

	// Create tunnel object
	tunnel := &Tunnel{
		ID:       subdomain,
		Conn:     conn,
		Requests: make(map[string]chan *ResponsePayload),
	}

	// Register the tunnel
	tunnelsMu.Lock()
	tunnels[subdomain] = tunnel
	tunnelsMu.Unlock()

	fmt.Printf("  ✅ New tunnel: %s\n", subdomain)

	// Tell the client their URL
	url := fmt.Sprintf("http://localhost:8080/%s", subdomain)
	payload, _ := json.Marshal(RegisteredPayload{
		URL:       url,
		Subdomain: subdomain,
	})
	conn.WriteJSON(Message{Type: "registered", Payload: payload})

	// Listen for messages from the client (responses)
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("  ❌ Tunnel %s disconnected\n", subdomain)
			break
		}

		// Handle response from client
		if msg.Type == "response" {
			var resp ResponsePayload
			json.Unmarshal(msg.Payload, &resp)

			// Find the waiting request and send the response
			tunnel.ReqMu.Lock()
			if ch, ok := tunnel.Requests[resp.ID]; ok {
				ch <- &resp
				delete(tunnel.Requests, resp.ID)
			}
			tunnel.ReqMu.Unlock()
		}
	}

	// Cleanup when client disconnects
	tunnelsMu.Lock()
	delete(tunnels, subdomain)
	tunnelsMu.Unlock()
	conn.Close()
}

// ============================================
// HTTP HANDLER (for public requests)
// ============================================

func handlePublicRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the path: /abc123/api/users -> subdomain=abc123, path=/api/users
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)

	// No subdomain provided
	if len(parts) == 0 || parts[0] == "" {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "tunl relay server\n\n")
		fmt.Fprintf(w, "To use a tunnel, visit: /<subdomain>/your/path\n")
		return
	}

	subdomain := parts[0]
	forwardPath := "/"
	if len(parts) > 1 {
		forwardPath = "/" + parts[1]
	}

	// Find the tunnel
	tunnelsMu.RLock()
	tunnel, exists := tunnels[subdomain]
	tunnelsMu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Tunnel '%s' not found", subdomain), http.StatusNotFound)
		return
	}

	// Read the request body
	body, _ := io.ReadAll(r.Body)

	// Copy headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create unique request ID
	reqID := generateID()

	// Create channel to wait for response
	respChan := make(chan *ResponsePayload, 1)
	tunnel.ReqMu.Lock()
	tunnel.Requests[reqID] = respChan
	tunnel.ReqMu.Unlock()

	// Build the request payload
	reqPayload, _ := json.Marshal(RequestPayload{
		ID:      reqID,
		Method:  r.Method,
		Path:    forwardPath,
		Headers: headers,
		Body:    body,
	})

	// Send request to tunnel client
	err := tunnel.Conn.WriteJSON(Message{Type: "request", Payload: reqPayload})
	if err != nil {
		http.Error(w, "Failed to forward request to tunnel", http.StatusBadGateway)
		return
	}

	fmt.Printf("  📥 %s %s -> %s\n", r.Method, forwardPath, subdomain)

	// Wait for response from tunnel client
	resp := <-respChan

	// Copy response headers
	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}

	// Write status code and body
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)

	fmt.Printf("  📤 %s %s <- %d\n", r.Method, forwardPath, resp.StatusCode)
}

// ============================================
// HELPERS
// ============================================

func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}