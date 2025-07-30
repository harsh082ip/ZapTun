# Zaptun 🚀

Zaptun is a powerful reverse tunneling tool, architecturally similar to services like ngrok or jprq, that exposes a local web server to the internet through a secure public URL. It's an effective way to test webhooks, demo projects without deploying, or access home servers from anywhere.

Built entirely in Go, this project serves as a practical exploration of advanced networking concepts, including TCP session management, reverse proxying, and high-performance connection multiplexing.

## Features

  * **HTTP Tunneling**: Expose any local HTTP server on a public-facing subdomain.
  * **Unique Subdomains**: Automatically generates a unique, random subdomain for each new session (e.g., `abcdef.zaptun.com`), preventing collisions.
  * **Concurrent Connections**: Built to handle a high volume of simultaneous HTTP requests efficiently through high-performance connection multiplexing.
  * **Connection Pooling**: The client uses a connection pool to communicate with the local service, eliminating TCP handshake overhead under load and preventing bottlenecks.
  * **Automatic Reconnects**: The client is resilient and will automatically attempt to re-establish a connection to the server if it is lost.
  * **Keep-Alive Heartbeats**: The client-server connection is kept alive using a heartbeat mechanism, preventing premature timeouts from network hardware or firewalls.

-----

## In-Depth Architecture

Zaptun's architecture is a client-server model designed to bypass NAT and firewall restrictions by reversing the typical connection flow. It consists of two primary applications: the **Zaptun Server** and the **Zaptun Client**.

### The Zaptun Server

The server is the public-facing anchor of the service, running on a cloud instance with a public IP. It operates on two distinct logical planes to separate concerns:

  * **Control Plane**: This plane listens on a dedicated TCP port (e.g., `:4443`). Its sole responsibility is to manage client sessions. When a Zaptun client connects, the control plane performs a handshake, assigns a unique subdomain, and establishes a multiplexed session. It maintains a **client registry**, which is a thread-safe map of all active clients and their sessions, acting as a routing table.

  * **Data Plane**: This is a public HTTP server listening on a standard port (e.g., `:80`). It's the entry point for all public traffic. When a request arrives, the data plane's reverse proxy logic inspects the `Host` header (e.g., `abcdef.zaptun.com`), uses the subdomain (`abcdef`) to look up the correct client session in the registry, and then forwards the request to that client through its tunnel.

### The Zaptun Client

The client is a lightweight command-line application run on a user's local machine. It initiates the "reverse" connection that makes the tunnel possible. Its duties include:

1.  Establishing a persistent TCP connection to the server's **Control Plane**.
2.  Wrapping this connection in a **multiplexed session** using `yamux`.
3.  Performing a handshake with the server to receive its assigned public URL.
4.  Listening for new data streams initiated by the server.
5.  For each stream (representing a public HTTP request), it forwards the data to the user's specified local web service (e.g., `localhost:8080`) using a connection from its local pool.

-----

## The Role and Importance of `yamux`

The most critical technical challenge in a tunneling service is handling multiple, simultaneous user requests over a single client-server TCP connection. A naive implementation would lead to data corruption as different HTTP requests get mixed together.

**`yamux` solves this by providing stream multiplexing.**

Think of the single TCP connection as a large, physical pipe. `yamux` allows us to create thousands of smaller, independent, virtual pipes inside that one physical pipe. Each virtual pipe is called a **stream**.

  * **Isolation**: Each HTTP request and its corresponding response are encapsulated in their own dedicated stream. Data from one request will never interfere with another, ensuring integrity.
  * **Concurrency**: Because the streams are independent, the server can process many requests in parallel without waiting for previous ones to complete. When the server gets a request for a client, it opens a new stream, sends the request, and can immediately start processing the next public request. The client will handle the streams as they arrive.
  * **Efficiency**: Creating a new `yamux` stream is extremely cheap and fast compared to establishing a new TCP connection. This allows the system to be highly responsive and scale to a large number of concurrent users.

Without a multiplexer like `yamux`, the service would be limited to handling one request at a time, making it impractical for real-world use.

-----

## End-to-End Request Flow

Here is a step-by-step walkthrough of what happens from client startup to a user seeing a webpage:

1.  **Client Starts**: The user runs `./zaptun-client --server zaptun.com:4443 --port 8080`.
2.  **Initial Connection**: The client opens a TCP connection to the server's control plane at `zaptun.com:4443`.
3.  **Session Establishment**: Both the client and server wrap this TCP connection in a `yamux` session.
4.  **Handshake**:
      * The client opens a new, dedicated "control stream" to the server.
      * The server accepts this control stream. It generates a unique ID (e.g., `abcdef`), adds the client's session to its registry (`"abcdef" -> session`), and sends the full public URL (`abcdef.zaptun.com`) back to the client over the control stream.
      * The client reads the URL and displays it to the user. The tunnel is now live.
5.  **Public Request**: A user on the internet navigates to `http://abcdef.zaptun.com`.
6.  **Server Receives Request**: The server's data plane (on port `:80`) receives the GET request. The `Host` header is `abcdef.zaptun.com`.
7.  **Routing**: The server's proxy extracts the subdomain `abcdef`, looks it up in the client registry, and finds the active `yamux` session for our client.
8.  **Forwarding via Stream**:
      * The server opens a new **data stream** over the client's `yamux` session.
      * It writes the full HTTP GET request into this new stream.
9.  **Client Receives Request**:
      * The client's `session.AcceptStream()` call unblocks, receiving the new data stream.
      * It launches a new goroutine to handle this stream.
      * It reads the HTTP request from the stream.
      * It gets a connection from its local **connection pool** and forwards the request to `localhost:8080`.
10. **Local Response**: The local web server processes the request and sends back an HTML response.
11. **Response Forwarding**:
      * The client reads the HTML response from `localhost:8080`.
      * It writes this response back into the **same data stream** it came from.
12. **Final Delivery**: The server reads the response from the data stream and forwards it back to the user's browser, which then renders the page.

-----

## Setup and Usage

### Prerequisites

  * Go 1.18 or higher
  * A server with a public IP address (to run the Zaptun server)
  * A registered domain name (e.g., `zaptun.com`)

### Configuration

1.  **DNS Setup**: In your domain registrar's control panel, you need to create a **wildcard DNS A record**. This is essential for routing all subdomain traffic to your server.

      * **Type**: `A`
      * **Name / Host**: `*`
      * **Value / Points to**: `YOUR_SERVER_IP_ADDRESS`

2.  **Server Configuration (`configs/server.json`)**:
    Update the `configs/server.json` file with your domain and desired port settings.

    ```json
    {
      "domain": "zaptun.com",
      "control_plane_addr": ":4443",
      "data_plane_addr": ":80",
      "log_file": "/var/log/zaptun/server.log",
      "log_level": "info"
    }
    ```

### Running the Service

1.  **Run the Server**:
    SSH into your public server, clone this repository, and run the server application.

    ```bash
    # Clone the repo
    git clone https://github.com/harsh082ip/zaptun.git
    cd zaptun

    # Run the server
    go run ./cmd/zaptun-server/ --config ./configs/server.json
    ```

2.  **Run the Client**:
    On your local machine, run the client application. Point it to your server's domain and specify the local port you want to expose.

    ```bash
    # Run the client to expose your local service running on port 8080
    go run ./cmd/zaptun-client/ --server zaptun.com:4443 --lp 8080
    ```

    The client will connect and display the public URL assigned by the server:
    `{"level":"info",...,"message":"Tunnel is live at: http://random-id.zaptun.com"}`

3.  **Access Your Service**:
    You can now access your local service from anywhere in the world by navigating to the provided public URL in your browser.

-----

## Code Structure

The project follows the standard Go project layout to maintain a clean separation of concerns.

  * `cmd/`: Contains the entry points for the two binaries produced by this project.
      * `zaptun-server/main.go`: The `main` function for the server application.
      * `zaptun-client/main.go`: The `main` function for the client application.
  * `internal/`: Holds the core application logic. This code is private to the project.
      * `server/`: All server-specific logic, including the control and data planes.
      * `client/`: All client-specific logic, including connecting, serving, and forwarding.
  * `pkg/`: Contains shared libraries that can be used by both the client and server (and potentially external applications).
      * `config/`: Logic for loading and parsing configuration files.
      * `logger/`: A shared logging setup for consistent output.

-----

## Future Improvements

  * **Serve static content**: Add support for serving static files
  * **Custom Subdomains**: Allow clients to request a specific subdomain via a command-line flag.
  * **Web Dashboard**: A status page served by the client to inspect traffic in real-time.

<a href="https://buymeacoffee.com/harshyt1975" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>
