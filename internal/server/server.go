package server

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/harsh082ip/ZapTun/config"
	log "github.com/harsh082ip/ZapTun/pkg/logger"
)

type Client struct {
	id   string // unique cname
	conn net.Conn
}

type Server struct {
	conf    *config.ServerConfig
	logger  *log.Logger
	clients map[string]*Client
	mutex   sync.RWMutex
}

func NewServer(conf *config.ServerConfig, logger *log.Logger) *Server {
	return &Server{
		conf:    conf,
		logger:  logger,
		clients: make(map[string]*Client),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.conf.ControlPlaneAddr)
	if err != nil {
		return err
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
		s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.logger.LogInfoMessage().Msg("New client connected...")

	// 1. Generate unique cname or client id
	var clientID string
	for {
		clientID = generateRandomCNAME(8)
		_, clientExists := s.clients[clientID]
		if !clientExists {
			break
		}
		s.logger.LogInfoMessage().Msgf("clientID: %v, already exists, attempting to create new one", clientID)
	}

	// 2. Register new client
	newClient := &Client{
		id:   clientID,
		conn: conn,
	}
	s.mutex.Lock()
	s.clients[clientID] = newClient
	s.mutex.Unlock()

	// 3. defer - proper cleanup for this client
	defer func() {
		s.mutex.Lock()
		delete(s.clients, clientID)
		s.mutex.Unlock()
		conn.Close()
		s.logger.LogInfoMessage().Msgf("Client: %v disconnected. Removed from registry", clientID)
	}()

	// 4. send assigned url back to the client
	assignedURL := fmt.Sprintf("%s.%s", clientID, s.conf.Domain)
	_, err := conn.Write([]byte(assignedURL + "\n"))
	if err != nil {
		s.logger.LogErrorMessage().Err(err).Msg("Failed to send assigned URL to client")
		return // This will trigger the deferred cleanup.
	}
	s.logger.LogInfoMessage().Msgf("Assigned URL %s to client %s", assignedURL, conn.RemoteAddr().String())

	// 5. Keep the connection alive.
	// We read from the connection in a loop. If the client disconnects,
	// the Read() will fail, the function will exit, and the deferred cleanup will run.
	// io.Copy(io.Discard, conn) is a simple way to do this.
	io.Copy(io.Discard, conn)

}
