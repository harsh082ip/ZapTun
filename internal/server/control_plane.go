package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/harsh082ip/ZapTun/internal/server/github"
	"github.com/harsh082ip/ZapTun/pkg/tunnel"
	"github.com/hashicorp/yamux"
)

func (s *Server) startControlPlane() {
	certPath := "cert.pem"
	keyPath := "privkey.pem"

	if s.conf.CertificatePath != "" && s.conf.PrivateKeyPath != "" {
		certPath = s.conf.CertificatePath
		keyPath = s.conf.PrivateKeyPath
	}

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

	// get access token from client
	var token string
	if err := json.NewDecoder(ctrlStream).Decode(&token); err != nil {
		s.logger.LogErrorMessage().Err(err).Msgf("failed to get Auth token from client")
		return
	}

	// validate auth token
	user, err := s.authenticator.Authenticate(token)
	if err != nil {
		s.logger.LogErrorMessage().Err(err).Msgf("failed to authenticate user")
		// also tell the client
		msg := fmt.Sprintf("authentication failed %s", "obtain auth token from http://zapyun.com/auth\n")
		if _, err := ctrlStream.Write([]byte(msg)); err != nil {
			s.logger.LogErrorMessage().Err(err).Msgf("failed to send auth error msg to the client")
		}
	}
	s.logger.LogInfoMessage().Msgf("allowd: %v", user.Allowed)

	if !user.Allowed {
		msg := fmt.Sprintf("authentication failed %s", "obtain auth token from http://zaptun.com/auth\n")
		if _, err := ctrlStream.Write([]byte(msg)); err != nil {
			s.logger.LogErrorMessage().Err(err).Msgf("failed to send auth error msg to the client")
		}
	}

	if _, err := ctrlStream.Write([]byte("auth_ok\n")); err != nil {
		s.logger.LogErrorMessage().Err(err).Msgf("failed to send auth success msg to client")
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
		s.handleHTTPTunnel(session, ctrlStream, &user)
	case "tcp":
		s.handleTCPTunnel(session, ctrlStream, &user)
	}
}

func (s *Server) handleHTTPTunnel(session *yamux.Session, ctrlStream net.Conn, user *github.User) {
	s.logger.LogInfoMessage().Msg("Handling HTTP tunnel request...")

	s.mutex.Lock()

	userRecord, exists := s.users[user.Login]
	if !exists {
		userRecord = &User{
			tunnels:   make(map[string]*Client),
			maxTunnel: 2,
		}
		s.users[user.Login] = userRecord
	}

	if len(userRecord.tunnels) >= userRecord.maxTunnel {
		s.mutex.Unlock()
		msg := fmt.Sprintf("err: max http tunnel limit reached (%d)", userRecord.maxTunnel)
		ctrlStream.Write([]byte(msg + "\n"))
		s.logger.LogWarnMessage().Msgf("Max tunnel limit reached for user: %v", user.Login)
		return
	}

	tunnelID := user.Login
	if _, idExists := userRecord.tunnels[tunnelID]; idExists {
		for i := 1; ; i++ {
			numberedID := fmt.Sprintf("%s-%d", user.Login, i)
			if _, idExists := userRecord.tunnels[numberedID]; !idExists {
				tunnelID = numberedID
				break
			}
		}
	}

	newClient := &Client{
		id:      tunnelID,
		session: session,
	}
	userRecord.tunnels[tunnelID] = newClient

	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		if userRec, ok := s.users[user.Login]; ok {
			delete(userRec.tunnels, tunnelID)
			if len(userRec.tunnels) == 0 {
				delete(s.users, user.Login)
			}
		}
		s.mutex.Unlock()
		session.Close()
		s.logger.LogInfoMessage().Msgf("Client tunnel %v disconnected. Removed from registry.", tunnelID)
	}()

	assignedURL := fmt.Sprintf("%s.%s", tunnelID, s.conf.Domain)
	if _, err := ctrlStream.Write([]byte(assignedURL + "\n")); err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return
	}
	s.logger.LogInfoMessage().Msgf("Assigned URL %s to user %s", assignedURL, user.Login)

	io.Copy(io.Discard, ctrlStream)
}

// handleTCPTunnel is now updated with fine-grained locking.
func (s *Server) handleTCPTunnel(session *yamux.Session, ctrlStream net.Conn, user *github.User) {
	s.logger.LogInfoMessage().Msg("Handling TCP tunnel request...")

	s.mutex.Lock()

	userRecord, exists := s.users[user.Login]
	if !exists {
		userRecord = &User{
			tunnels:   make(map[string]*Client),
			maxTunnel: 2,
		}
		s.users[user.Login] = userRecord
	}

	if len(userRecord.tunnels) >= userRecord.maxTunnel {
		s.mutex.Unlock() // Unlock before returning
		msg := fmt.Sprintf("err: max tcp tunnel limit reached (%d)", userRecord.maxTunnel)
		ctrlStream.Write([]byte(msg + "\n"))
		s.logger.LogWarnMessage().Msgf("Max tunnel limit reached for user: %v", user.Login)
		return
	}

	// Get the port number and release the lock
	port := s.nextTCPPort
	s.nextTCPPort++

	s.mutex.Unlock()

	publicAddr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", publicAddr)
	if err != nil {
		ctrlStream.Write([]byte("error: could not allocate public port\n"))
		return
	}
	s.logger.LogInfoMessage().Msgf("TCP tunnel for %s listening on %s", user.Login, publicAddr)

	tunnelID := fmt.Sprintf("tcp-%s-%d", user.Login, len(userRecord.tunnels)+1)
	newClient := &Client{
		id:       tunnelID,
		session:  session,
		listener: listener,
	}

	s.mutex.Lock()
	s.users[user.Login].tunnels[tunnelID] = newClient
	s.mutex.Unlock()

	// Defer cleanup
	defer func() {
		s.mutex.Lock()
		if userRec, ok := s.users[user.Login]; ok {
			delete(userRec.tunnels, tunnelID)
			if len(userRec.tunnels) == 0 {
				delete(s.users, user.Login)
			}
		}
		s.mutex.Unlock()
		session.Close()
		listener.Close()
		s.logger.LogInfoMessage().Msgf("Client tunnel %v disconnected. Closed public listener on %s.", tunnelID, publicAddr)
	}()

	publicURL := fmt.Sprintf("%s:%d", s.conf.Domain, port)
	if _, err := ctrlStream.Write([]byte(publicURL + "\n")); err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return
	}

	go s.proxyTCP(listener, session)
	io.Copy(io.Discard, ctrlStream)
}

// proxyTCP accepts public connections and forwards them to the client via yamux streams.
func (s *Server) proxyTCP(listener net.Listener, session *yamux.Session) {
	for {
		// Accept a new connection from the public internet
		publicConn, err := listener.Accept()
		if err != nil {
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
