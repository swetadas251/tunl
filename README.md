<div align="center">



\# tunl



\*\*Expose your localhost to the internet in seconds.\*\*



\[Features](#features) • \[Installation](#installation) • \[Usage](#usage) • \[How It Works](#how-it-works)



!\[Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat\&logo=go\&logoColor=white)

!\[License](https://img.shields.io/badge/License-MIT-green)

!\[Platform](https://img.shields.io/badge/Platform-Windows%20|%20macOS%20|%20Linux-lightgrey)



</div>



---



\## What is tunl?



tunl is a lightweight CLI tool that creates a secure tunnel from the public internet to your local machine. Share your local development server with anyone, anywhere.



\*\*Example:\*\*

```bash

$ tunl 3000



&nbsp; TUNNEL IS LIVE!



&nbsp; Public URL:  https://tunl-npt8.onrender.com/a1b2c3d4

&nbsp; Forwards to: http://localhost:3000

```



Now anyone can access your local server at that public URL!



---



\## Features



\- \*\*Public URLs\*\* — Get a real HTTPS URL for your localhost

\- \*\*Fast\*\* — Built in Go for minimal latency

\- \*\*Secure\*\* — All traffic encrypted with TLS

\- \*\*Free\*\* — No account or payment required

\- \*\*Cross-platform\*\* — Works on Windows, macOS, and Linux



---



\## Installation



\### Download Binary



Download the latest release for your platform:



\- \[Windows (64-bit)](https://github.com/swetadas251/tunl/releases/latest)

\- \[macOS (64-bit)](https://github.com/swetadas251/tunl/releases/latest)

\- \[Linux (64-bit)](https://github.com/swetadas251/tunl/releases/latest)



\### Build from Source

```bash

git clone https://github.com/swetadas251/tunl.git

cd tunl

go build -o tunl ./cmd/tunl

```



---



\## Usage



\### Basic Usage



Expose port 3000:

```bash

tunl 3000

```



\### Expose Any Port

```bash

tunl 8080    # Expose port 8080

tunl 5173    # Expose Vite dev server

tunl 4200    # Expose Angular dev server

```



---



\## How It Works

┌─────────────────────────────────────────────────────────────────┐

│                         INTERNET                                │

│                            │                                    │

│                            ▼                                    │

│                   ┌─────────────────┐                           │

│                   │  Relay Server   │                           │

│                   │   (Render.com)  │                           │

│                   └────────┬────────┘                           │

│                            │                                    │

│                            │ WebSocket (TLS)                    │

│                            │                                    │

└────────────────────────────┼────────────────────────────────────┘

│

▼

┌─────────────────┐

│   tunl client   │

│  (your machine) │

└────────┬────────┘

│

│ HTTP

│

▼

┌─────────────────┐

│   Your App      │

│  localhost:3000 │

└─────────────────┘



1\. You run `tunl 3000` on your machine

2\. Client connects to relay server via WebSocket

3\. Relay assigns you a unique public URL

4\. When someone visits that URL, the request goes to the relay

5\. Relay forwards it through the WebSocket to your client

6\. Client forwards it to your local server

7\. Response travels back the same way



---



\## Tech Stack



| Component | Technology |

|-----------|------------|

| CLI Client | Go |

| Relay Server | Go |

| Communication | WebSocket |

| Hosting | Render.com |

| Protocol | JSON over WebSocket |



---



\## Project Structure

tunl/

├── cmd/

│   ├── tunl/           # CLI client

│   │   └── main.go

│   └── relay/          # Relay server

│       └── main.go

├── internal/

│   └── protocol/       # Shared types

│       └── messages.go

├── go.mod

└── README.md



---



\## Self-Hosting



Want to run your own relay server?

```bash

\# Clone the repo

git clone https://github.com/swetadas251/tunl.git

cd tunl



\# Build the relay

go build -o relay ./cmd/relay



\# Run it

PORT=8080 ./relay

```



Then point your client to your relay:

```bash

tunl 3000 wss://your-server.com/tunnel

```



---



\## Contributing



Contributions are welcome! Feel free to:



\- Report bugs

\- Suggest features

\- Submit pull requests



---



\## License



MIT License - feel free to use this for anything!



---



<div align="center">



\*\*Built by \[Sweta Das](https://github.com/swetadas251)\*\*





</div>

