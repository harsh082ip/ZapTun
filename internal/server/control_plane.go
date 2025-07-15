package server

import (
	"fmt"
	"io"
	"net"

	"github.com/hashicorp/yamux"
)

func (s *Server) startControlPlane() {
	listener, err := net.Listen("tcp", s.conf.ControlPlaneAddr)
	if err != nil {
		s.logger.LogErrorMessage().Msgf("failed to start control plane on: %v, err: %+v", s.conf.ControlPlaneAddr, err)
		return
	}
	defer listener.Close()

	s.logger.LogInfoMessage().Msgf("Starting Control Plane on: %v", s.conf.ControlPlaneAddr)
	for {
		conn, err := listener.Accept()
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

	//  Generate unique cname or client id
	var clientID string
	for {
		clientID = generateRandomCNAME(8)
		_, clientExists := s.clients[clientID]
		if !clientExists {
			break
		}
		s.logger.LogInfoMessage().Msgf("clientID: %v, already exists, attempting to create new one", clientID)
	}

	// Register new client
	newClient := &Client{
		id:      clientID,
		session: session,
	}
	s.mutex.Lock()
	s.clients[clientID] = newClient
	s.mutex.Unlock()

	// defer - proper cleanup for this client
	defer func() {
		s.mutex.Lock()
		delete(s.clients, clientID)
		s.mutex.Unlock()
		session.Close()
		s.logger.LogInfoMessage().Msgf("Client: %v disconnected. Removed from registry", clientID)
	}()

	// send assigned url back to the client
	assignedURL := fmt.Sprintf("%s.%s", clientID, s.conf.Domain)
	_, err = ctrlStream.Write([]byte(assignedURL + "\n"))
	if err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return // This will trigger the deferred cleanup.
	}
	s.logger.LogInfoMessage().Msgf("Assigned URL %s to client %s", assignedURL, conn.RemoteAddr().String())

	// 5. Keep the connection alive.
	// We read from the connection in a loop. If the client disconnects,
	// the Read() will fail, the function will exit, and the deferred cleanup will run.
	// io.Copy(io.Discard, conn) is a simple way to do this.
	io.Copy(io.Discard, ctrlStream)

}
