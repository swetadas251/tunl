package protocol

import "encoding/json"

// Message is the envelope for all client-server communication.
// Payload is kept as raw JSON so each side can unmarshal it
// into the appropriate type based on the message Type.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// RegisteredPayload is sent by the relay server to confirm a tunnel is active.
type RegisteredPayload struct {
	URL       string `json:"url"`       // Full public URL, e.g. "https://tunl-npt8.onrender.com/a1b2c3d4"
	Subdomain string `json:"subdomain"` // Just the ID part, e.g. "a1b2c3d4"
}

// RequestPayload represents an HTTP request that needs to be forwarded to localhost.
type RequestPayload struct {
	ID      string            `json:"id"`      // Unique ID to match request with response
	Method  string            `json:"method"`  // HTTP method: GET, POST, etc.
	Path    string            `json:"path"`    // URL path: /api/users
	Headers map[string]string `json:"headers"` // HTTP headers
	Body    []byte            `json:"body"`    // Request body (for POST, PUT, etc.)
}

// ResponsePayload is the HTTP response from localhost going back to the requester.
type ResponsePayload struct {
	ID         string            `json:"id"`          // Matches the request ID
	StatusCode int               `json:"status_code"` // HTTP status: 200, 404, 500, etc.
	Headers    map[string]string `json:"headers"`     // Response headers
	Body       []byte            `json:"body"`        // Response body
}

// ErrorPayload is sent when something goes wrong.
type ErrorPayload struct {
	Message string `json:"message"`
}
