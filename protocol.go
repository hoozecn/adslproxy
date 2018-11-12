package adslproxy

import (
	"fmt"
	"net"
	"time"
)

const HeartbeatInterval = 5 * time.Second

const ReadTimeout = 6 * time.Second

const WriteTimeout = 6 * time.Second

const ClientConnectTimeout = 5 * time.Second

const Reconnect = "adslproxy-reconnect"

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

// Forward indicates the forward mapping
type Forward struct {
	// Name of the forward
	Name string
	// Left is the addr that server listens on
	Left *net.TCPAddr
	// Right is the addr that agent listens on
	Right string

	Options string

	listener *net.TCPListener
}

func (f *Forward) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%s [%s=>%s]", f.Name, f.Left, f.Right)
}
