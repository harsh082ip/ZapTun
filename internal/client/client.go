package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/hashicorp/yamux"
)

type Client struct {
	serverAddr string
	localPort  int
	logger     *logger.Logger
}

func NewClient(serverAddr string, localPort int, log *logger.Logger) (*Client, error) {
	return &Client{
		serverAddr: serverAddr,
		localPort:  localPort,
		logger:     log,
	}, nil
}

func (c *Client) Start() error {
	c.logger.LogInfoMessage().Msgf("Connecting to server at %s", c.serverAddr)
	c.logger.LogInfoMessage().Msgf("Will forward traffic to localhost:%d", c.localPort)
	for {
		if err := c.connectAndServe(); err != nil {
			c.logger.LogErrorMessage().Err(err).Msg("Connection error. Retrying in 5 seconds...")
		}
		time.Sleep(5 * time.Second)
	}
	// return nil
}

// connectAndServe handles a single connection lifecycle.
func (c *Client) connectAndServe() error {
	// Connect to the server's control plane.
	conn, err := net.Dial("tcp", c.serverAddr)
	if err != nil {
		c.logger.LogErrorMessage().Msgf("failed to connect to control plane server, err: %+v", err)
		return fmt.Errorf("failed to connect to control plane server, err: %+v", err)
	}
	defer conn.Close()

	// yamux config
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = false
	// yamuxConfig.KeepAliveInterval = 60 * time.Hour
	// yamuxConfig.ConnectionWriteTimeout = 15 * time.Second

	// Establish a yamux session over the TCP connection.
	session, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		c.logger.LogErrorMessage().Msgf("failed to establish a yamux session over tcp connection, err: %+v", err)
		return fmt.Errorf("failed to establish a yamux session over tcp connection, err: %+v", err)
	}
	defer session.Close()

	// Open a "control stream" to the server for the handshake.
	ctrlStream, err := session.OpenStream()
	if err != nil {
		c.logger.LogErrorMessage().Msgf("failed to open control stream to the server, err: %+v", err)
		return fmt.Errorf("failed to open control stream to the server, err: %+v", err)
	}
	defer ctrlStream.Close()

	// Read the assigned public URL from the server.
	assignedURL, err := bufio.NewReader(ctrlStream).ReadString('\n')
	if err != nil {
		c.logger.LogErrorMessage().Msgf("failed to read assigned url from server, err: %+v", err)
		return fmt.Errorf("failed to read assigned url from server, err: %+v", err)
	}
	cleaned := strings.TrimSuffix(assignedURL, "\n")
	assignedURL = cleaned
	c.logger.LogInfoMessage().Msgf("Tunnel is live at: http://%s", assignedURL)

	// Start listening for new streams from the server (for proxied requests)
	for {
		// This blocks until the server opens a new stream for a public request.
		proxyStream, err := session.AcceptStream()
		if err != nil {
			// This error means the session is likely closed.
			c.logger.LogErrorMessage().Msgf("failed to read stream from server, err: %v", err)
			return err
		}
		// Handle each proxied request in its own goroutine.
		go c.handleProxyStream(proxyStream)
	}
}

// handleProxyStream forwards a proxied request to the local service.
func (c *Client) handleProxyStream(proxyStream net.Conn) {
	defer proxyStream.Close()
	c.logger.LogInfoMessage().Msg("Accepted new stream from server")

	// Read the HTTP request from the server.
	req, err := http.ReadRequest(bufio.NewReader(proxyStream))
	if err != nil {
		c.logger.LogErrorMessage().Msgf("failed to read http request from server, err: %+v", err)
		return
	}

	// Dial a new connection to the local service.
	localServiceConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", c.localPort))
	if err != nil {
		c.logger.LogErrorMessage().Err(err).Msgf("Failed to connect to local service on port %d", c.localPort)
		// Inform the user's browser that the local service is down.
		resp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("Local service unavailable")),
		}
		resp.Write(proxyStream)
		return
	}
	defer localServiceConn.Close()

	//   Write the request to the local service.
	if err := req.Write(localServiceConn); err != nil {
		c.logger.LogErrorMessage().Err(err).Msg("Failed to write request to local service")
		return
	}

	// Copy the response from the local service back to the proxy stream.
	_, err = io.Copy(proxyStream, localServiceConn)
	if err != nil {
		c.logger.LogErrorMessage().Err(err).Msg("Failed to copy response from local service to proxy stream")
	}
}
