package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/hashicorp/yamux"
)

func (s *Server) startControlPlane() {
	certPath := "cert.pem"
	keyPath := "privkey.pem"

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		s.logger.LogInfoMessage().Msgf("Failed to load TLS certificate, err:%v", err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := net.Listen("tcp", s.conf.ControlPlaneAddr)
	if err != nil {
		s.logger.LogErrorMessage().Msgf("failed to start control plane on: %v, err: %+v", s.conf.ControlPlaneAddr, err)
		return
	}
	tlsListener := tls.NewListener(listener, tlsConfig)
	defer tlsListener.Close()

	s.logger.LogInfoMessage().Msgf("Starting Control Plane on: %v", s.conf.ControlPlaneAddr)
	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			s.logger.LogErrorMessage().Msgf("failed to accept connection on control plane, err: %+v", err)
			continue
		}

		// handle net.conn
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.logger.LogInfoMessage().Msg("New client connected...")
	fmt.Printf("%s", conn.RemoteAddr())

	// yamux config
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = false
	// yamuxConfig.KeepAliveInterval = 60 * time.Hour
	// yamuxConfig.ConnectionWriteTimeout = 15 * time.Second

	// wrap raw TCP into a yamux session
	session, err := yamux.Server(conn, yamuxConfig)
	if err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to create yamux session")
		return
	}

	// The client is expected to open a stream. We'll wait for it.
	// This stream will be used for control messages in the future.
	// For now, accepting it proves the session is working.
	ctrlStream, err := session.AcceptStream()
	if err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to accept control stream")
		session.Close()
		return
	}

	// decode tunnel type from ctrlstream
	var msg tunnel.ControlMessage
	if err := json.NewDecoder(ctrlStream).Decode(&msg); err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to decode control message")
		return
	}

	switch msg.Type {
	case "http":
		s.handleHTTPTunnel(conn, session, ctrlStream)
	case "tcp":
		s.handleTCPTunnel(conn, session, ctrlStream)
	}
}

// handleHTTPTunnel contains the logic for setting up an HTTP tunnel.
func (s *Server) handleHTTPTunnel(conn net.Conn, session *yamux.Session, ctrlStream net.Conn) {
	s.logger.LogInfoMessage().Msg("Handling HTTP tunnel request...")

	// Generate unique subdomain
	var clientID string
	for {
		clientID = generateRandomSubdomain(8)
		s.mutex.RLock()
		_, clientExists := s.clients[clientID]
		s.mutex.RUnlock()
		if !clientExists {
			break
		}
	}

	newClient := &Client{
		id:      clientID,
		session: session,
	}
	s.mutex.Lock()
	s.clients[clientID] = newClient
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		delete(s.clients, clientID)
		s.mutex.Unlock()
		session.Close()
		s.logger.LogInfoMessage().Msgf("Client: %v disconnected. Removed from registry", clientID)
	}()

	assignedURL := fmt.Sprintf("%s.%s", clientID, s.conf.Domain)
	if _, err := ctrlStream.Write([]byte(assignedURL + "\n")); err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return
	}
	s.logger.LogInfoMessage().Msgf("Assigned URL %s to client %s", assignedURL, conn.RemoteAddr().String())

	// Block to keep the control stream alive and detect disconnection
	io.Copy(io.Discard, ctrlStream)
}

func (s *Server) handleTCPTunnel(conn net.Conn, session *yamux.Session, ctrlStream net.Conn) {
	s.logger.LogInfoMessage().Msg("Handling TCP tunnel request...")

	// 1. Allocate a new port for this client
	// will make port allocation logic more robust
	s.mutex.Lock()
	port := s.nextTCPPort
	s.nextTCPPort++
	s.mutex.Unlock()

	publicAddr := fmt.Sprintf("0.0.0.0:%d", port)

	// 2. Start a new public TCP listener on the allocated port
	listener, err := net.Listen("tcp", publicAddr)
	if err != nil {
		s.logger.LogErrorMessage().Msgf("Failed to start public listener on %s, err: %+v", publicAddr, err)
		// Inform the client that the setup failed
		ctrlStream.Write([]byte("error: could not allocate public port\n"))
		return
	}

	s.logger.LogInfoMessage().Msgf("TCP tunnel listening for public connections on %s", publicAddr)

	// 3. Register the client, now including its listener
	clientID := generateRandomSubdomain(8)
	newClient := &Client{
		id:       clientID,
		session:  session,
		listener: listener, // Store the listener
	}
	s.mutex.Lock()
	s.clients[clientID] = newClient
	s.mutex.Unlock()

	// 4. Defer cleanup
	defer func() {
		s.mutex.Lock()
		delete(s.clients, clientID)
		s.mutex.Unlock()
		session.Close()
		listener.Close()
		s.logger.LogInfoMessage().Msgf("Client %v disconnected. Closed public listener on %s.", clientID, publicAddr)
	}()

	// Inform the client of its public address
	publicURL := fmt.Sprintf("%s:%d", s.conf.Domain, port)
	if _, err := ctrlStream.Write([]byte(publicURL + "\n")); err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return
	}

	// Start the proxying loop for the new public listener
	go s.proxyTCP(listener, session)

	// keep the control stream(zaptun-client) alive
	io.Copy(io.Discard, ctrlStream)
}

// proxyTCP accepts public connections and forwards them to the client via yamux streams.
func (s *Server) proxyTCP(listener net.Listener, session *yamux.Session) {
	for {
		// Accept a new connection from the public internet
		publicConn, err := listener.Accept()
		if err != nil {
			// This error likely means the listener was closed, so we can exit.
			s.logger.LogWarnMessage().Err(err).Msg("Public TCP listener failed to accept")
			return
		}

		s.logger.LogInfoMessage().Msgf("Accepted new public TCP connection from %s", publicConn.RemoteAddr())

		// For each public connection, open a new stream to the client
		proxyStream, err := session.OpenStream()
		if err != nil {
			s.logger.LogErrorMessage().Err(err).Msg("Failed to open yamux stream for TCP proxy")
			publicConn.Close()
			continue
		}

		// copy the data concurrently
		go func() {
			defer proxyStream.Close()
			defer publicConn.Close()
			io.Copy(proxyStream, publicConn)
		}()
		go func() {
			defer proxyStream.Close()
			defer publicConn.Close()
			io.Copy(publicConn, proxyStream)
		}()
	}
}
