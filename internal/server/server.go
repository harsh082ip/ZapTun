package server

import (
	"net"
	"sync"

	"github.com/harsh082ip/ZapTun/config"
	"github.com/harsh082ip/ZapTun/internal/server/github"
	log "github.com/harsh082ip/ZapTun/pkg/logger"
	"github.com/hashicorp/yamux"
)

type Client struct {
	id       string // unique subdomain
	session  *yamux.Session
	listener net.Listener
}

type User struct {
	tunnels   map[string]*Client
	maxTunnel int
}

type Server struct {
	conf          *config.ServerConfig
	logger        *log.Logger
	users         map[string]*User
	mutex         sync.RWMutex
	nextTCPPort   int
	authenticator github.Authenticator
}

func NewServer(conf *config.ServerConfig, logger *log.Logger, oauth github.Authenticator) *Server {
	return &Server{
		conf:          conf,
		logger:        logger,
		users:         make(map[string]*User),
		nextTCPPort:   30000, // will change port allocation logic in future PRs
		authenticator: oauth,
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
