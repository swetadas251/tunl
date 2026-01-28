package protocol

// MessageType identifies what kind of message this is
type MessageType string

const (
	// Client -> Server messages
	TypeRegister MessageType = "register"  // Client wants to create a tunnel
	TypeResponse MessageType = "response"  // Client sending HTTP response back

	// Server -> Client messages
	TypeRegistered MessageType = "registered" // Server confirms tunnel is ready
	TypeRequest    MessageType = "request"    // Server forwarding an HTTP request
	TypeError      MessageType = "error"      // Something went wrong
)

// Message is the envelope for all client-server communication
type Message struct {
	Type    MessageType `json:"type"`
	Payload any         `json:"payload"`
}

// RegisterPayload - sent by client to request a new tunnel
type RegisterPayload struct {
	// Empty for now, but could add auth token later
}

// RegisteredPayload - sent by server to confirm tunnel is active
type RegisteredPayload struct {
	URL       string `json:"url"`       // Full public URL, e.g. "https://abc123.tunl.dev"
	Subdomain string `json:"subdomain"` // Just the subdomain part, e.g. "abc123"
}

// RequestPayload - an HTTP request that needs to be forwarded to localhost
type RequestPayload struct {
	ID      string            `json:"id"`      // Unique ID to match request with response
	Method  string            `json:"method"`  // HTTP method: GET, POST, etc.
	Path    string            `json:"path"`    // URL path: /api/users
	Headers map[string]string `json:"headers"` // HTTP headers
	Body    []byte            `json:"body"`    // Request body (for POST, PUT, etc.)
}

// ResponsePayload - the HTTP response from localhost going back to the requester
type ResponsePayload struct {
	ID         string            `json:"id"`          // Matches the request ID
	StatusCode int               `json:"status_code"` // HTTP status: 200, 404, 500, etc.
	Headers    map[string]string `json:"headers"`     // Response headers
	Body       []byte            `json:"body"`        // Response body
}

// ErrorPayload - sent when something goes wrong
type ErrorPayload struct {
	Message string `json:"message"`
}