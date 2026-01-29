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
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type RegisteredPayload struct {
	URL       string `json:"url"`
	Subdomain string `json:"subdomain"`
}

type RequestPayload struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

type ResponsePayload struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

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

	relayURL := "ws://localhost:8080/tunnel"
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
		fmt.Println("no (warning)")
	}

	fmt.Printf("  Connecting to relay... ")
	conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		fmt.Println("failed")
		fmt.Printf("\n  Error: %v\n", err)
		fmt.Println("\n  Make sure the relay server is running:")
		fmt.Println("    ./bin/relay.exe")
		os.Exit(1)
	}
	fmt.Println("connected")

	conn.WriteJSON(Message{Type: "register", Payload: nil})

	var msg Message
	err = conn.ReadJSON(&msg)
	if err != nil || msg.Type != "registered" {
		fmt.Println("  Registration failed")
		os.Exit(1)
	}

	var registered RegisteredPayload
	json.Unmarshal(msg.Payload, &registered)

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
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("\n  Connection lost: %v\n", err)
			break
		}

		if msg.Type == "request" {
			var req RequestPayload
			json.Unmarshal(msg.Payload, &req)
			go handleRequest(conn, localTarget, req)
		}
	}
}

func handleRequest(conn *websocket.Conn, localTarget string, req RequestPayload) {
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

	payload, _ := json.Marshal(ResponsePayload{
		ID:         req.ID,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	})
	conn.WriteJSON(Message{Type: "response", Payload: payload})

	printRequest(req.Method, req.Path, resp.StatusCode, time.Since(startTime))
}

func sendErrorResponse(conn *websocket.Conn, reqID string, status int, message string) {
	payload, _ := json.Marshal(ResponsePayload{
		ID:         reqID,
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(message),
	})
	conn.WriteJSON(Message{Type: "response", Payload: payload})
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