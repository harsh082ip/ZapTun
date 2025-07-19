package server

import (
	"net"
	"sync"

	"github.com/harsh082ip/ZapTun/config"
	log "github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/hashicorp/yamux"
)

type Client struct {
	id       string // unique subdomain
	session  *yamux.Session
	listener net.Listener
}

type Server struct {
	conf        *config.ServerConfig
	logger      *log.Logger
	clients     map[string]*Client
	mutex       sync.RWMutex
	nextTCPPort int
}

func NewServer(conf *config.ServerConfig, logger *log.Logger) *Server {
	return &Server{
		conf:        conf,
		logger:      logger,
		clients:     make(map[string]*Client),
		nextTCPPort: 30000, // will change port allocation logic in future PRs
	}
}

func (s *Server) Start() error {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.startControlPlane()
	}()

	go func() {
		defer wg.Done()
		s.startDataPlane()
	}()

	s.logger.LogInfoMessage().Msg("Server started succesfully. Waiting for connections...")
	wg.Wait()
	return nil
}
