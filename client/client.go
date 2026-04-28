package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/smallnest/doubletake/game"
)

// Client is a TCP client that connects to the game server.
type Client struct {
	conn     net.Conn
	messages chan game.Message
	done     chan struct{}
	closeOnce sync.Once
}

// NewClient creates a new Client.
func NewClient() *Client {
	return &Client{
		messages: make(chan game.Message, 64),
		done:     make(chan struct{}),
	}
}

// Connect establishes a TCP connection to the server at the given address.
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("unable to connect to server: %w", err)
	}
	c.conn = conn
	go c.receiveLoop()
	return nil
}

// Send encodes and sends a message to the server.
func (c *Client) Send(msg game.Message) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	data := game.Encode(msg)
	_, err := c.conn.Write([]byte(data))
	if err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}

// Messages returns the channel on which received messages are delivered.
func (c *Client) Messages() <-chan game.Message {
	return c.messages
}

// Disconnect closes the connection and stops the receive loop.
// It is safe to call Disconnect() multiple times.
func (c *Client) Disconnect() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			c.conn.Close()
		}
		close(c.messages)
	})
}

// receiveLoop reads messages from the server in a background goroutine.
// When the server closes the connection (EOF), the loop exits gracefully.
func (c *Client) receiveLoop() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := scanner.Text()
		msg, err := game.Decode(line)
		if err != nil {
			continue
		}
		select {
		case c.messages <- msg:
		case <-c.done:
			return
		}
	}
	// Scanner stopped (EOF or error) — signal disconnect
	c.Disconnect()
}
