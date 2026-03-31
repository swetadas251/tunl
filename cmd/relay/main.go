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
	"time"

	"github.com/gorilla/websocket"
	"github.com/swetadas251/tunl/internal/protocol"
)

type Tunnel struct {
	ID       string
	Conn     *websocket.Conn
	WriteMu  sync.Mutex // protects concurrent WebSocket writes
	Requests map[string]chan *protocol.ResponsePayload
	ReqMu    sync.Mutex
}

var tunnels = make(map[string]*Tunnel)
var tunnelsMu sync.RWMutex

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
	http.HandleFunc("/health", handleHealthCheck)
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

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
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
		Requests: make(map[string]chan *protocol.ResponsePayload),
	}

	tunnelsMu.Lock()
	tunnels[subdomain] = tunnel
	tunnelsMu.Unlock()

	fmt.Printf("  New tunnel: %s\n", subdomain)

	var url string
	renderURL := os.Getenv("RENDER_EXTERNAL_URL")
	if renderURL != "" {
		url = fmt.Sprintf("%s/%s", renderURL, subdomain)
	} else {
		url = fmt.Sprintf("http://localhost:8080/%s", subdomain)
	}

	payload, _ := json.Marshal(protocol.RegisteredPayload{
		URL:       url,
		Subdomain: subdomain,
	})

	tunnel.WriteMu.Lock()
	conn.WriteJSON(protocol.Message{Type: "registered", Payload: payload})
	tunnel.WriteMu.Unlock()

	for {
		var msg protocol.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("  Tunnel %s disconnected\n", subdomain)
			break
		}

		if msg.Type == "response" {
			var resp protocol.ResponsePayload
			if err := json.Unmarshal(msg.Payload, &resp); err != nil {
				log.Printf("  Failed to parse response from tunnel %s: %v", subdomain, err)
				continue
			}

			tunnel.ReqMu.Lock()
			if ch, ok := tunnel.Requests[resp.ID]; ok {
				ch <- &resp
				delete(tunnel.Requests, resp.ID)
			}
			tunnel.ReqMu.Unlock()
		}
	}

	// Clean up: remove tunnel and close any pending request channels
	tunnelsMu.Lock()
	delete(tunnels, subdomain)
	tunnelsMu.Unlock()

	tunnel.ReqMu.Lock()
	for id, ch := range tunnel.Requests {
		close(ch)
		delete(tunnel.Requests, id)
	}
	tunnel.ReqMu.Unlock()

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

	respChan := make(chan *protocol.ResponsePayload, 1)
	tunnel.ReqMu.Lock()
	tunnel.Requests[reqID] = respChan
	tunnel.ReqMu.Unlock()

	reqPayload, _ := json.Marshal(protocol.RequestPayload{
		ID:      reqID,
		Method:  r.Method,
		Path:    forwardPath,
		Headers: headers,
		Body:    body,
	})

	tunnel.WriteMu.Lock()
	err := tunnel.Conn.WriteJSON(protocol.Message{Type: "request", Payload: reqPayload})
	tunnel.WriteMu.Unlock()

	if err != nil {
		tunnel.ReqMu.Lock()
		delete(tunnel.Requests, reqID)
		tunnel.ReqMu.Unlock()
		http.Error(w, "Failed to forward request to tunnel", http.StatusBadGateway)
		return
	}

	fmt.Printf("  %s %s -> %s\n", r.Method, forwardPath, subdomain)

	// Wait for response with a timeout to prevent goroutine leaks
	select {
	case resp, ok := <-respChan:
		if !ok {
			// Channel was closed — tunnel disconnected
			http.Error(w, "Tunnel disconnected", http.StatusBadGateway)
			return
		}

		for key, value := range resp.Headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)

		fmt.Printf("  %s %s <- %d\n", r.Method, forwardPath, resp.StatusCode)

	case <-time.After(30 * time.Second):
		tunnel.ReqMu.Lock()
		delete(tunnel.Requests, reqID)
		tunnel.ReqMu.Unlock()
		http.Error(w, "Tunnel response timed out", http.StatusGatewayTimeout)
		fmt.Printf("  %s %s <- TIMEOUT\n", r.Method, forwardPath)
	}
}

func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}