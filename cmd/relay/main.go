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

// Tunnel represents one connected client
type Tunnel struct {
	ID       string
	Conn     *websocket.Conn
	Requests map[string]chan *ResponsePayload
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

// Store all active tunnels
var tunnels = make(map[string]*Tunnel)
var tunnelsMu sync.RWMutex

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/tunnel", handleTunnelConnection)
	http.HandleFunc("/", handlePublicRequest)

	fmt.Println("==================================================")
	fmt.Println("  tunl relay server")
	fmt.Println("==================================================")
	fmt.Printf("  HTTP Server:  http://localhost:%s\n", port)
	fmt.Printf("  WebSocket:    ws://localhost:%s/tunnel\n", port)
	fmt.Println("==================================================")
	fmt.Println("  Waiting for tunnel clients to connect...")
	fmt.Println("")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleTunnelConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	subdomain := generateID()

	tunnel := &Tunnel{
		ID:       subdomain,
		Conn:     conn,
		Requests: make(map[string]chan *ResponsePayload),
	}

	tunnelsMu.Lock()
	tunnels[subdomain] = tunnel
	tunnelsMu.Unlock()

	fmt.Printf("  New tunnel: %s\n", subdomain)

	// Build the public URL
	var url string
	renderURL := os.Getenv("RENDER_EXTERNAL_URL")
	if renderURL != "" {
		// Running on Render
		url = fmt.Sprintf("%s/%s", renderURL, subdomain)
	} else {
		// Running locally
		url = fmt.Sprintf("http://localhost:8080/%s", subdomain)
	}

	payload, _ := json.Marshal(RegisteredPayload{
		URL:       url,
		Subdomain: subdomain,
	})
	conn.WriteJSON(Message{Type: "registered", Payload: payload})

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("  Tunnel %s disconnected\n", subdomain)
			break
		}

		if msg.Type == "response" {
			var resp ResponsePayload
			json.Unmarshal(msg.Payload, &resp)

			tunnel.ReqMu.Lock()
			if ch, ok := tunnel.Requests[resp.ID]; ok {
				ch <- &resp
				delete(tunnel.Requests, resp.ID)
			}
			tunnel.ReqMu.Unlock()
		}
	}

	tunnelsMu.Lock()
	delete(tunnels, subdomain)
	tunnelsMu.Unlock()
	conn.Close()
}

func handlePublicRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)

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

	tunnelsMu.RLock()
	tunnel, exists := tunnels[subdomain]
	tunnelsMu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Tunnel '%s' not found", subdomain), http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(r.Body)

	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	reqID := generateID()

	respChan := make(chan *ResponsePayload, 1)
	tunnel.ReqMu.Lock()
	tunnel.Requests[reqID] = respChan
	tunnel.ReqMu.Unlock()

	reqPayload, _ := json.Marshal(RequestPayload{
		ID:      reqID,
		Method:  r.Method,
		Path:    forwardPath,
		Headers: headers,
		Body:    body,
	})

	err := tunnel.Conn.WriteJSON(Message{Type: "request", Payload