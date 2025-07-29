package client

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"
)

type Client struct {
	serverAddr string
	localPort  int
	controlMsg *tunnel.ControlMessage
	conf       *config.ClientConfig
	logLevel   zerolog.Level
	logger     *logger.Logger
}

func NewClient(controlMsg *tunnel.ControlMessage, conf *config.ClientConfig, localPort int, log *logger.Logger) (*Client, error) {
	return &Client{
		serverAddr: conf.Remote.ServerAddr,
		localPort:  localPort,
		controlMsg: controlMsg,
		logger:     log,
		conf:       conf,
		logLevel:   zerolog.Disabled,
	}, nil
}

func (c *Client) Start(logLevel zerolog.Level) error {
	c.logger.LogInfoMessage().Msgf("Connecting to server at %s", c.serverAddr)
	c.logger.LogInfoMessage().Msgf("Will forward traffic to localhost:%d", c.localPort)
	c.logLevel = logLevel
	for {
		if err := c.connectAndServe(); err != nil {
			c.logger.LogErrorMessage().Err(err).Msg("Connection error. Retrying in 5 seconds...")
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) connectAndServe() error {
	tlsConfig := &tls.Config{
		// InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", c.serverAddr, tlsConfig)
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

	// send access token
	err = json.NewEncoder(ctrlStream).Encode(c.conf.Local.AuthToken)
	if err != nil {
		return fmt.Errorf("failed to send control message: %w", err)
	}

	authResp, err := bufio.NewReader(ctrlStream).ReadString('\n')
	if err != nil {
		fmt.Printf("failed to read auth response: %v", err)
		os.Exit(1)
	}
	authResp = strings.TrimSpace(authResp)
	if authResp != "auth_ok" {
		fmt.Printf("%s\n", authResp)
		os.Exit(1)
	}

	err = json.NewEncoder(ctrlStream).Encode(c.controlMsg)
	if err != nil {
		return fmt.Errorf("failed to send control message: %w", err)
	}
	// log.Println(c.controlMsg)

	response, err := bufio.NewReader(ctrlStream).ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response from server: %w", err)
	}
	response = strings.TrimSpace(response)
	iserrorMsg := strings.Contains(response, "err:")
	if iserrorMsg {
		fmt.Println(response)
		os.Exit(1)
	}

	if c.controlMsg.Type == "http" {
		if c.logLevel == zerolog.Disabled {
			fmt.Print("\033[H\033[2J") // clear
			fmt.Printf("Status: \t Online \n")
			fmt.Printf("Protocol: \t %s \n", strings.ToUpper(c.controlMsg.Type))
			fmt.Printf("Forwarding: \t %s -> %s \n",
				fmt.Sprintf("https://%s", response),
				fmt.Sprintf("http://localhost:%d", c.localPort))
		}
		c.logger.LogInfoMessage().Msgf("Tunnel is live at: http://%s", response)

	} else {
		if c.logLevel == zerolog.Disabled {
			fmt.Print("\033[H\033[2J") // clear
			fmt.Printf("Status: \t Online \n")
			fmt.Printf("Protocol: \t %s \n", strings.ToUpper(c.controlMsg.Type))
			fmt.Printf("Forwarding:\t %s -> %s\n",
				fmt.Sprintf("tcp://%s", response),
				fmt.Sprintf("tcp://localhost:%d", c.localPort),
			)

		}
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
	if c.logLevel == zerolog.Disabled {
		fmt.Printf("Incoming: \t %s \n", proxyStream.RemoteAddr())
	}
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
