package signalflow

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// How long to wait between connections in case of a bad connection.
var ReconnectDelay = 5 * time.Second

type wsConn struct {
	sync.Mutex
	ctx       context.Context
	streamURL *url.URL

	outgoingTextMsgs   chan *outgoingMessage
	incomingTextMsgs   chan []byte
	incomingBinaryMsgs chan []byte
	readErrCh          chan error
	readCh             chan struct{}
	connectedCh        chan struct{}

	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	PostDisconnectCallback func()
	PostConnectMessage     func() []byte
}

type outgoingMessage struct {
	bytes    []byte
	resultCh chan error
}

func newWebsocketConn(ctx context.Context, streamURL *url.URL) *wsConn {
	ws := &wsConn{
		ctx:                ctx,
		streamURL:          streamURL,
		incomingTextMsgs:   make(chan []byte),
		incomingBinaryMsgs: make(chan []byte),
		outgoingTextMsgs:   make(chan *outgoingMessage),
		readErrCh:          make(chan error),
		readCh:             make(chan struct{}),
		connectedCh:        make(chan struct{}),
		ReadTimeout:        1 * time.Minute,
		WriteTimeout:       20 * time.Second,
	}
	return ws
}

func (c *wsConn) IncomingTextMessages() <-chan []byte {
	return c.incomingTextMsgs
}

func (c *wsConn) IncomingBinaryMessages() <-chan []byte {
	return c.incomingBinaryMsgs
}

func (c *wsConn) OutgoingTextMessages() chan<- *outgoingMessage {
	return c.outgoingTextMsgs
}

func (c *wsConn) ConnectedSignal() <-chan struct{} {
	return c.connectedCh
}

// Run keeps the connection alive and puts all incoming messages into a channel
// as needed.
func (c *wsConn) Run() {
	var conn *websocket.Conn

	for {
		if conn == nil {
			// This will get run on before the first connection as well.
			if c.PostDisconnectCallback != nil {
				c.PostDisconnectCallback()
			}

			var err error
			conn, err = c.connect()
			if err != nil {
				log.Printf("Error connecting to SignalFlow websocket: %v", err)
				time.Sleep(ReconnectDelay)
				continue
			}

			err = c.postConnect(conn)
			if err != nil {
				log.Printf("Error setting up SignalFlow websocket: %v", err)
				conn.Close()
				conn = nil
				time.Sleep(ReconnectDelay)
				continue
			}

			go c.readNextMessage(conn)
		}

		select {
		case <-c.ctx.Done():
			conn.Close()
			return
		case <-c.readCh:
			go c.readNextMessage(conn)
		case err := <-c.readErrCh:
			log.Printf("Error reading from SignalFlow websocket: %v", err)
			conn.Close()
			conn = nil
			time.Sleep(ReconnectDelay)
		case msg := <-c.outgoingTextMsgs:
			err := c.writeMessage(conn, msg.bytes)
			msg.resultCh <- err
			if err != nil {
				// Force the connection closed if it isn't already and wait for
				// the read goroutine to finish.
				conn.Close()
				select {
				case <-c.readErrCh:
				case <-c.readCh:
				}
				conn = nil
				time.Sleep(ReconnectDelay)
			}
		}
	}
}

func (c *wsConn) connect() (*websocket.Conn, error) {
	connectURL := *c.streamURL
	connectURL.Path = path.Join(c.streamURL.Path, "connect")
	conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, connectURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not connect Signalflow websocket: %v", err)
	}
	return conn, nil
}

func (c *wsConn) postConnect(conn *websocket.Conn) error {
	if c.PostConnectMessage != nil {
		msg := c.PostConnectMessage()
		if msg != nil {
			return c.writeMessage(conn, msg)
		}
	}
	return nil
}

func (c *wsConn) readNextMessage(conn *websocket.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(c.ReadTimeout)); err != nil {
		c.readErrCh <- fmt.Errorf("could not set read timeout in SignalFlow client: %v", err)
		return
	}

	typ, bytes, err := conn.ReadMessage()
	if err != nil {
		c.readErrCh <- err
		return
	}
	if typ == websocket.TextMessage {
		c.incomingTextMsgs <- bytes
	} else {
		c.incomingBinaryMsgs <- bytes
	}
	c.readCh <- struct{}{}
}

func (c *wsConn) writeMessage(conn *websocket.Conn, msgBytes []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return fmt.Errorf("could not set write timeout for SignalFlow request: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		return err
	}
	return nil
}
