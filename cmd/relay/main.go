package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/swetadas251/tunl/internal/protocol"
)

// writeMu protects concurrent writes to the WebSocket connection.
// gorilla/websocket only supports one concurrent writer at a time.
var writeMu sync.Mutex

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tunl <port> [relay-url]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  tunl 3000")
		fmt.Println("  tunl 8080 ws://localhost:8080/tunnel")
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil || port < 1 || port > 65535 {
		fmt.Printf("Invalid port: %s\n", os.Args[1])
		os.Exit(1)
	}

	relayURL := "wss://tunl-npt8.onrender.com/tunnel"
	if len(os.Args) > 2 {
		relayURL = os.Args[2]
	}

	localTarget := fmt.Sprintf("http://localhost:%d", port)

	fmt.Println("")
	fmt.Println("==================================================")
	fmt.Println("  tunl client")
	fmt.Println("==================================================")
	fmt.Printf("  Local server:  %s\n", localTarget)
	fmt.Printf("  Relay server:  %s\n", relayURL)
	fmt.Println("==================================================")
	fmt.Println("")

	fmt.Printf("  Checking if localhost:%d is reachable... ", port)
	if isLocalServerRunning(port) {
		fmt.Println("yes")
	} else {
		fmt.Println("no (warning: start your server before sending requests)")
	}

	fmt.Printf("  Connecting to relay (may take ~30s on first connect)... ")
	dialer := websocket.Dialer{
		HandshakeTimeout: 60 * time.Second,
	}
	conn, _, err := dialer.Dial(relayURL, nil)
	if err != nil {
		fmt.Println("failed")
		fmt.Printf("\n  Error: %v\n", err)
		fmt.Println("\n  Make sure the relay server is running.")
		fmt.Println("  For local testing: go run ./cmd/relay")
		os.Exit(1)
	}
	fmt.Println("connected")

	writeMu.Lock()
	conn.WriteJSON(protocol.Message{Type: "register", Payload: nil})
	writeMu.Unlock()

	var msg protocol.Message
	err = conn.ReadJSON(&msg)
	if err != nil || msg.Type != "registered" {
		fmt.Println("  Registration failed")
		os.Exit(1)
	}

	var registered protocol.RegisteredPayload
	if err := json.Unmarshal(msg.Payload, &registered); err != nil {
		fmt.Printf("  Failed to parse registration response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("==================================================")
	fmt.Println("  TUNNEL IS LIVE!")
	fmt.Println("==================================================")
	fmt.Println("")
	fmt.Printf("  Public URL:  %s\n", registered.URL)
	fmt.Printf("  Forwards to: %s\n", localTarget)
	fmt.Println("")
	fmt.Println("==================================================")
	fmt.Println("  Press Ctrl+C to stop the tunnel")
	fmt.Println("==================================================")
	fmt.Println("")
	fmt.Println("  Requests:")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n  Tunnel closed. Goodbye!")
		conn.Close()
		os.Exit(0)
	}()

	for {
		var msg protocol.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("\n  Connection lost: %v\n", err)
			break
		}

		if msg.Type == "request" {
			var req protocol.RequestPayload
			if err := json.Unmarshal(msg.Payload, &req); err != nil {
				fmt.Printf("  Failed to parse request: %v\n", err)
				continue
			}
			go handleRequest(conn, localTarget, req)
		}
	}
}

func handleRequest(conn *websocket.Conn, localTarget string, req protocol.RequestPayload) {
	startTime := time.Now()
	localURL := localTarget + req.Path

	httpReq, err := http.NewRequest(req.Method, localURL, bytes.NewReader(req.Body))
	if err != nil {
		sendErrorResponse(conn, req.ID, 500, "Failed to create request")
		printRequest(req.Method, req.Path, 500, time.Since(startTime))
		return
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		sendErrorResponse(conn, req.ID, 502, "Could not reach local server")
		printRequest(req.Method, req.Path, 502, time.Since(startTime))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	payload, _ := json.Marshal(protocol.ResponsePayload{
		ID:         req.ID,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	})

	writeMu.Lock()
	conn.WriteJSON(protocol.Message{Type: "response", Payload: payload})
	writeMu.Unlock()

	printRequest(req.Method, req.Path, resp.StatusCode, time.Since(startTime))
}

func sendErrorResponse(conn *websocket.Conn, reqID string, status int, message string) {
	payload, _ := json.Marshal(protocol.ResponsePayload{
		ID:         reqID,
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(message),
	})

	writeMu.Lock()
	conn.WriteJSON(protocol.Message{Type: "response", Payload: payload})
	writeMu.Unlock()
}

func printRequest(method, path string, status int, duration time.Duration) {
	fmt.Printf("  %s %s -> %d (%dms)\n", method, path, status, duration.Milliseconds())
}

func isLocalServerRunning(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
