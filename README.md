# tunl

**Expose your localhost to the internet in seconds.**

A lightweight CLI tool that creates secure tunnels from the public internet to your local machine. Built with Go and WebSockets.

---

## Quick Start
```bash
./tunl 3000
```
```
TUNNEL IS LIVE!

Public URL:  https://tunl-npt8.onrender.com/a1b2c3d4
Forwards to: http://localhost:3000
```

Now anyone can access your local server at that public URL!

---

## Features

- **Public URLs** — Get a real HTTPS URL for your localhost
- **Fast** — Built in Go for minimal latency  
- **Secure** — All traffic encrypted with TLS
- **Free** — No account or payment required
- **Cross-platform** — Works on Windows, macOS, and Linux

---

## Installation
```bash
git clone https://github.com/swetadas251/tunl.git
cd tunl
go build -o tunl ./cmd/tunl
```

---

## Usage
```bash
# Expose port 3000
./tunl 3000

# Expose port 8080
./tunl 8080
```

---

## How It Works
```
Internet Request
       |
       v
+-------------+
| Relay Server| (Render.com)
+-------------+
       |
       | WebSocket (TLS)
       v
+-------------+
| tunl client | (your machine)
+-------------+
       |
       | HTTP
       v
+-------------+
|  Your App   | (localhost:3000)
+-------------+
```

1. You run `tunl 3000` on your machine
2. Client connects to relay server via WebSocket
3. Relay assigns you a unique public URL
4. Requests to that URL get forwarded through the tunnel to your localhost

---

## Tech Stack

- **Language:** Go
- **Communication:** WebSocket
- **Hosting:** Render.com
- **Protocol:** JSON over WebSocket

---

## Project Structure
```
tunl/
├── cmd/
│   ├── tunl/        # CLI client
│   └── relay/       # Relay server
├── internal/
│   └── protocol/    # Shared types
├── go.mod
└── README.md
```

---

## License

MIT License

---

**Built by [Sweta Das](https://github.com/swetadas251)**

