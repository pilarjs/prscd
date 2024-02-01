package chirp

import (
	"net"
	"sync"

	"github.com/gobwas/ws/wsutil"
	"github.com/quic-go/quic-go"
)

// Connection is connection either WebSocket or WebTransport
type Connection interface {
	// RemoteAddr returns the client network address.
	RemoteAddr() string
	// Write the data to the connection
	Write(msg []byte) error
	// RawWrite write the raw bytes to the connection, this is a low-level implementation
	RawWrite(buf []byte) (int, error)
}

/*** WebSocket ***/

// NewWebSocketConnection creates a new WebSocketConnection
func NewWebSocketConnection(conn net.Conn) Connection {
	return &WebSocketConnection{
		underlyingConn: conn,
	}
}

// WebSocketConnection is a WebSocket connection
type WebSocketConnection struct {
	mu             sync.Mutex
	underlyingConn net.Conn
}

// RemoteAddr returns the client network address.
func (c *WebSocketConnection) RemoteAddr() string {
	return (c.underlyingConn).RemoteAddr().String()
}

// Write the data to the connection
func (c *WebSocketConnection) Write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return wsutil.WriteServerBinary(c.underlyingConn, msg)
}

// RawWrite write the raw bytes to the connection, this is a low-level implementation
func (c *WebSocketConnection) RawWrite(buf []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.underlyingConn.Write(buf)
}

/*** WebTransport ***/

// NewWebTransportConnection creates a new WebTransportConnection
func NewWebTransportConnection(conn quic.Connection) Connection {
	return &WebTransportConnection{
		underlyingConn: conn,
	}
}

// WebTransportConnection is a WebTransport connection
type WebTransportConnection struct {
	mu             sync.Mutex
	underlyingConn quic.Connection
}

// RemoteAddr returns the client network address.
func (c *WebTransportConnection) RemoteAddr() string {
	return c.underlyingConn.RemoteAddr().String()
}

// Write the data to the connection
func (c *WebTransportConnection) Write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// add 0x00 to msg
	buf := []byte{0x00}
	buf = append(buf, msg...)
	if err := c.underlyingConn.SendDatagram(buf); err != nil {
		log.Error("SendMessage error", "remote", c.RemoteAddr(), "err", err)
		return err
	}
	return nil
}

// RawWrite write the raw bytes to the connection, this is a low-level implementation
func (c *WebTransportConnection) RawWrite(buf []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.underlyingConn.SendDatagram(buf); err != nil {
		log.Error("SendMessage error", "remote", c.RemoteAddr(), "err", err)
		return 0, err
	}
	return len(buf), nil
}
