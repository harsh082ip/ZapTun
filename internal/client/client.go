package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/hashicorp/yamux"
)

type Client struct {
	serverAddr string
	localPort  int
	controlMsg *tunnel.ControlMessage
	logger     *logger.Logger
}

func NewClient(serverAddr string, controlMsg *tunnel.ControlMessage, localPort int, log *logger.Logger) (*Client, error) {
	return &Client{
		serverAddr: serverAddr,
		localPort:  localPort,
		controlMsg: controlMsg,
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
}

func (c *Client) connectAndServe() error {
	conn, err := net.Dial("tcp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to control plane server: %w", err)
	}
	defer conn.Close()

	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = false

	session, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		return fmt.Errorf("failed to establish yamux session: %w", err)
	}
	defer session.Close()

	ctrlStream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("failed to open control stream: %w", err)
	}
	defer ctrlStream.Close()

	err = json.NewEncoder(ctrlStream).Encode(c.controlMsg)
	if err != nil {
		return fmt.Errorf("failed to send control message: %w", err)
	}

	response, err := bufio.NewReader(ctrlStream).ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response from server: %w", err)
	}
	response = strings.TrimSpace(response)

	if c.controlMsg.Type == "http" {
		c.logger.LogInfoMessage().Msgf("Tunnel is live at: http://%s", response)
	} else {
		c.logger.LogInfoMessage().Msgf("Tunnel is live at: %s", response)
	}

	for {
		proxyStream, err := session.AcceptStream()
		if err != nil {
			return err
		}
		go c.handleProxyStream(proxyStream, c.controlMsg.Type)
	}
}

func (c *Client) handleProxyStream(proxyStream net.Conn, tunnelType string) {
	defer proxyStream.Close()
	c.logger.LogInfoMessage().Msgf("Accepted new %s stream from server", tunnelType)

	localServiceConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", c.localPort))
	if err != nil {
		c.logger.LogErrorMessage().Err(err).Msgf("Failed to connect to local service on port %d", c.localPort)
		if tunnelType == "http" {
			resp := &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("Local service unavailable")),
			}
			resp.Write(proxyStream)
		}
		return
	}

	defer localServiceConn.Close()

	switch tunnelType {
	case "http":
		req, err := http.ReadRequest(bufio.NewReader(proxyStream))
		if err != nil {
			c.logger.LogErrorMessage().Err(err).Msg("Failed to read http request from server")
			return
		}

		if err := req.Write(localServiceConn); err != nil {
			c.logger.LogErrorMessage().Err(err).Msg("Failed to write request to local service")
			return
		}

		io.Copy(proxyStream, localServiceConn)

	case "tcp":
		go func() {
			io.Copy(localServiceConn, proxyStream)
		}()
		io.Copy(proxyStream, localServiceConn)
	}
}
