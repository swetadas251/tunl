# tunl

**Expose your localhost to the internet in seconds.**

A lightweight CLI tool that creates secure tunnels from the public internet to your local machine — like a minimal [ngrok](https://ngrok.com), built from scratch in Go.

---

## Quick Start

```bash
git clone https://github.com/swetadas251/tunl.git
cd tunl
go build -o tunl ./cmd/tunl
./tunl 3000
```

```
==================================================
  TUNNEL IS LIVE!
==================================================

  Public URL:  https://tunl-npt8.onrender.com/a1b2c3d4
  Forwards to: http://localhost:3000

==================================================
  Press Ctrl+C to stop the tunnel
==================================================
```

Now anyone on the internet can access your local server at that URL.

> **Note:** The relay is hosted on Render's free tier, so the first connection may take ~30 seconds to wake up. Subsequent connections are instant.

---

## Why I Built This

I wanted to understand how tools like ngrok work under the hood — not just use them, but build the core mechanics from scratch.

The result is a two-part system: a **relay server** that accepts incoming HTTP requests and a **CLI client** that opens a persistent WebSocket connection to the relay. When a request hits the relay, it gets serialized as JSON, pushed through the WebSocket to the client, forwarded to your local server, and the response travels back the same path. Each request gets a unique ID so responses are matched correctly, even under concurrent load.

Key concepts I worked with: WebSocket-based bidirectional communication, HTTP request/response multiplexing over a single connection, concurrent request handling with goroutines and channels, and graceful connection lifecycle management.

---

## How It Works

```
    Internet Request
           │
           ▼
   ┌───────────────┐
   │  Relay Server  │  (Render.com)
   └───────┬───────┘
           │ WebSocket (TLS)
           ▼
   ┌───────────────┐
   │  tunl client   │  (your machine)
   └───────┬───────┘
           │ HTTP
           ▼
   ┌───────────────┐
   │   Your App     │  (localhost:3000)
   └───────────────┘
```

1. You run `tunl 3000` on your machine
2. The client opens a WebSocket connection to the relay server
3. The relay assigns a unique public URL and sends it back
4. Any HTTP request to that URL gets forwarded through the WebSocket tunnel to your localhost
5. Your local server's response is sent back through the tunnel to the original requester

---

## Try It Locally (no deployment needed)

You can test the entire system on your machine without the hosted relay:

```bash
# Terminal 1 — Start any local server
python3 -m http.server 3000

# Terminal 2 — Start the relay server locally
go run ./cmd/relay

# Terminal 3 — Connect the tunnel client to the local relay
go run ./cmd/tunl 3000 ws://localhost:8080/tunnel
# Note the subdomain in the output, e.g. "a1b2c3d4"

# Terminal 4 — Test it
curl http://localhost:8080/a1b2c3d4/
# You should see the response from your python server
```

---

## Features

- **Public URLs** — Get a real HTTPS URL for your localhost
- **Fast** — Built in Go for minimal latency
- **Secure** — All traffic encrypted with TLS via the hosted relay
- **Free** — No account or payment required
- **Cross-platform** — Works on Windows, macOS, and Linux

---

## Project Structure

```
tunl/
├── cmd/
│   ├── tunl/           # CLI client — connects to relay, forwards traffic to localhost
│   └── relay/          # Relay server — accepts public HTTP, forwards via WebSocket
├── internal/
│   └── protocol/       # Shared message types used by both client and server
├── go.mod
└── README.md
```

---

## Tech Stack

- **Language:** Go
- **Transport:** WebSocket ([gorilla/websocket](https://github.com/gorilla/websocket))
- **Hosting:** Render.com (free tier)
- **Protocol:** JSON messages over WebSocket with request/response ID correlation

---

## License

MIT — see [LICENSE.txt](./LICENSE.txt)

---

**Built by [Sweta Das](https://github.com/swetadas251)**
