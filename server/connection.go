package server

import (
	"io"
	"log"
	"net"

	"github.com/BaptTF/sickgnal-server/handlers"
	"github.com/BaptTF/sickgnal-server/protocol"
)

// Connection represents a single client connection.
type Connection struct {
	conn    net.Conn
	writer  *protocol.ConnWriter
	handler *handlers.Handler
	userID  string // Set after authentication
	token   string // Set after authentication
}

// NewConnection creates a new connection handler.
func NewConnection(conn net.Conn, handler *handlers.Handler) *Connection {
	return &Connection{
		conn:    conn,
		writer:  protocol.NewConnWriter(conn),
		handler: handler,
	}
}

// Run processes messages from the client until disconnection.
func (c *Connection) Run() {
	defer func() {
		// Unregister from instant relay on disconnect
		if c.userID != "" {
			c.handler.Relay().Unregister(c.userID)
		}
		c.conn.Close()
		log.Printf("Client disconnected: %s", c.conn.RemoteAddr())
	}()

	log.Printf("Client connected: %s", c.conn.RemoteAddr())

	for {
		pkt, err := protocol.ReadPacket(c.conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error from %s: %v", c.conn.RemoteAddr(), err)
			}
			return
		}

		c.processPacket(pkt)
	}
}

func (c *Connection) processPacket(pkt *protocol.Packet) {
	msg, ty, err := protocol.ParseMessage(pkt.Message)
	if err != nil {
		log.Printf("Parse error from %s: %v", c.conn.RemoteAddr(), err)
		// Check if it's a client-only message type (0-10)
		if ty != "" {
			switch ty {
			case protocol.TyPreKeyBundle,
				protocol.TyConversationOpen,
				protocol.TyConversationMessage,
				protocol.TyKeyRotation,
				protocol.TyUserProfile:
				c.sendError(pkt.RequestID, protocol.ErrMessageTypeNotAccepted)
				return
			}
		}
		c.sendError(pkt.RequestID, protocol.ErrInvalidMessage)
		return
	}

	ctx := &handlers.Context{
		RequestID: pkt.RequestID,
		Writer:    c.writer,
		UserID:    c.userID,
		Token:     c.token,
		SetAuth: func(userID, token string) {
			c.userID = userID
			c.token = token
		},
		ConnWriter: c.writer,
		RawMessage: pkt.Message,
	}

	response := c.handler.Handle(ctx, msg, ty)
	if response != nil {
		if err := c.writer.WritePacket(pkt.RequestID, response); err != nil {
			log.Printf("Write error to %s: %v", c.conn.RemoteAddr(), err)
		}
	}
}

func (c *Connection) sendError(requestID uint16, code protocol.ErrorCode) {
	if err := c.writer.WritePacket(requestID, protocol.NewError(code)); err != nil {
		log.Printf("Write error to %s: %v", c.conn.RemoteAddr(), err)
	}
}
