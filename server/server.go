package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"github.com/BaptTF/sickgnal-server/config"
	"github.com/BaptTF/sickgnal-server/handlers"
	"gorm.io/gorm"
)

// Server is the main TCP/TLS server.
type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	handler *handlers.Handler
}

// New creates a new server instance.
func New(cfg *config.Config, db *gorm.DB) *Server {
	return &Server{
		cfg:     cfg,
		db:      db,
		handler: handlers.New(db),
	}
}

// Run starts the server and listens for connections.
func (s *Server) Run() error {
	listener, err := s.createListener()
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("Server listening on %s (TLS: %v)", s.cfg.ListenAddr(), s.cfg.TLSEnabled())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) createListener() (net.Listener, error) {
	if s.cfg.TLSEnabled() {
		cert, err := tls.LoadX509KeyPair(s.cfg.TLSCert, s.cfg.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load TLS cert/key: %w", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		return tls.Listen("tcp", s.cfg.ListenAddr(), tlsCfg)
	}
	return net.Listen("tcp", s.cfg.ListenAddr())
}

func (s *Server) handleConnection(conn net.Conn) {
	c := NewConnection(conn, s.handler)
	c.Run()
}
