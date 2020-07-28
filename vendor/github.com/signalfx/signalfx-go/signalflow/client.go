package signalflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/signalfx/signalfx-go/signalflow/messages"
)

// Client for SignalFlow via websockets (SSE is not currently supported).
type Client struct {
	// Access token for the org
	token                  string
	userAgent              string
	defaultMetadataTimeout time.Duration
	nextChannelNum         int64
	conn                   *wsConn
	readTimeout            time.Duration
	// How long to wait for writes to the websocket to finish
	writeTimeout   time.Duration
	streamURL      *url.URL
	channelsByName map[string]*Channel
	outgoingCh     chan *clientMessageRequest

	ctx    context.Context
	cancel context.CancelFunc
	sync.Mutex
}

type clientMessageRequest struct {
	msg      interface{}
	resultCh chan error
}

// ClientParam is the common type of configuration functions for the SignalFlow client
type ClientParam func(*Client) error

// StreamURL lets you set the full URL to the stream endpoint, including the
// path.
func StreamURL(streamEndpoint string) ClientParam {
	return func(c *Client) error {
		var err error
		c.streamURL, err = url.Parse(streamEndpoint)
		return err
	}
}

// StreamURLForRealm can be used to configure the websocket url for a specific
// SignalFx realm.
func StreamURLForRealm(realm string) ClientParam {
	return func(c *Client) error {
		var err error
		c.streamURL, err = url.Parse(fmt.Sprintf("wss://stream.%s.signalfx.com/v2/signalflow", realm))
		return err
	}
}

// AccessToken can be used to provide a SignalFx organization access token or
// user access token to the SignalFlow client.
func AccessToken(token string) ClientParam {
	return func(c *Client) error {
		c.token = token
		return nil
	}
}

// UserAgent allows setting the `userAgent` field when authenticating to
// SignalFlow.  This can be useful for accounting how many jobs are started
// from each client.
func UserAgent(userAgent string) ClientParam {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

// MetadataTimeout is the default amount of time that calls to metadata
// accessors on a SignalFlow Computation instance will wait to receive the
// metadata from the backend before failing and returning a zero value. Usually
// metadata comes in very quickly from the stream after the job start.
func MetadataTimeout(timeout time.Duration) ClientParam {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("MetadataTimeout cannot be <= 0")
		}
		c.defaultMetadataTimeout = timeout
		return nil
	}
}

// ReadTimeout sets the duration to wait between messages that come on the
// websocket.  If the resolution of the job is very low, this should be
// increased.
func ReadTimeout(timeout time.Duration) ClientParam {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("ReadTimeout cannot be <= 0")
		}
		c.readTimeout = timeout
		return nil
	}
}

// WriteTimeout sets the maximum duration to wait to send a single message when
// writing messages to the SignalFlow server over the WebSocket connection.
func WriteTimeout(timeout time.Duration) ClientParam {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("WriteTimeout cannot be <= 0")
		}
		c.writeTimeout = timeout
		return nil
	}
}

// NewClient makes a new SignalFlow client that will immediately try and
// connect to the SignalFlow backend.
func NewClient(options ...ClientParam) (*Client, error) {
	c := &Client{
		streamURL: &url.URL{
			Scheme: "wss",
			Host:   "stream.us0.signalfx.com",
			Path:   "/v2/signalflow",
		},
		readTimeout:            1 * time.Minute,
		writeTimeout:           5 * time.Second,
		channelsByName:         make(map[string]*Channel),
		defaultMetadataTimeout: 5 * time.Second,
		outgoingCh:             make(chan *clientMessageRequest),
	}

	for i := range options {
		if err := options[i](c); err != nil {
			return nil, err
		}
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.conn = newWebsocketConn(c.ctx, c.streamURL)
	c.conn.ReadTimeout = c.readTimeout
	c.conn.WriteTimeout = c.writeTimeout
	c.conn.PostDisconnectCallback = func() {
		c.closeRegisteredChannels()
	}

	c.conn.PostConnectMessage = func() []byte {
		bytes, err := c.makeAuthRequest()
		if err != nil {
			// This could almost be a panic
			log.Printf("Could not make auth request: %v", err)
			return nil
		}
		return bytes
	}

	go c.conn.Run()
	go c.run()

	return c, nil
}

func (c *Client) newUniqueChannelName() string {
	name := fmt.Sprintf("ch-%d", atomic.AddInt64(&c.nextChannelNum, 1))
	return name
}

// Writes all messages from a single goroutine since that is required by
// websocket library.
func (c *Client) run() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.conn.IncomingTextMessages():
			err := c.handleMessage(msg, websocket.TextMessage)
			if err != nil {
				log.Printf("Error handling SignalFlow text message: %v", err)
			}
		case msg := <-c.conn.IncomingBinaryMessages():
			err := c.handleMessage(msg, websocket.BinaryMessage)
			if err != nil {
				log.Printf("Error handling SignalFlow binary message: %v", err)
			}
		case outMsg := <-c.outgoingCh:
			outMsg.resultCh <- c.serializeAndWriteMessage(outMsg.msg)
		}
	}
}

func (c *Client) sendMessage(message interface{}) error {
	resultCh := make(chan error, 1)
	c.outgoingCh <- &clientMessageRequest{
		msg:      message,
		resultCh: resultCh,
	}
	return <-resultCh
}

func (c *Client) serializeMessage(message interface{}) ([]byte, error) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("could not marshal SignalFlow request: %v", err)
	}
	return msgBytes, nil
}

func (c *Client) serializeAndWriteMessage(message interface{}) error {
	msgBytes, err := c.serializeMessage(message)
	if err != nil {
		return err
	}

	resultCh := make(chan error, 1)
	c.conn.OutgoingTextMessages() <- &outgoingMessage{
		bytes:    msgBytes,
		resultCh: resultCh,
	}
	return <-resultCh
}

func (c *Client) handleMessage(msgBytes []byte, msgTyp int) error {
	message, err := messages.ParseMessage(msgBytes, msgTyp == websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("could not parse SignalFlow message: %v", err)
	}

	if cm, ok := message.(messages.ChannelMessage); ok {
		channelName := cm.Channel()
		c.Lock()
		channel, ok := c.channelsByName[channelName]
		if !ok {
			// The channel should have existed before, but now doesn't,
			// probably because it was closed.
			return nil
		} else if channelName == "" {
			c.acceptMessage(message)
			return nil
		}
		channel.AcceptMessage(message)
		c.Unlock()
	} else {
		return c.acceptMessage(message)
	}
	return nil
}

// acceptMessages accepts non-channel specific messages.  The only one that I
// know of is the authenticated response.
func (c *Client) acceptMessage(message messages.Message) error {
	if _, ok := message.(*messages.AuthenticatedMessage); ok {
		return nil
	} else if msg, ok := message.(*messages.BaseJSONMessage); ok {
		data := msg.RawData()
		if data != nil && data["event"] == "KEEP_ALIVE" {
			// Ignore keep alive messages
			return nil
		}
	}

	return fmt.Errorf("unknown SignalFlow message received: %v", message)
}

// Sends the authenticate message but does not wait for a response.
func (c *Client) makeAuthRequest() ([]byte, error) {
	return c.serializeMessage(&AuthRequest{
		Token:     c.token,
		UserAgent: c.userAgent,
	})
}

// Execute a SignalFlow job and return a channel upon which informational
// messages and data will flow.
func (c *Client) Execute(req *ExecuteRequest) (*Computation, error) {
	if req.Channel == "" {
		req.Channel = c.newUniqueChannelName()
	}

	err := c.sendMessage(req)
	if err != nil {
		return nil, err
	}

	return newComputation(c.ctx, c.registerChannel(req.Channel), c), nil
}

// Stop sends a job stop request message to the backend.  It does not wait for
// jobs to actually be stopped.
func (c *Client) Stop(req *StopRequest) error {
	// We are assuming that the stop request will always come from the same
	// client that started it with the Execute method above, and thus the
	// connection is still active (i.e. we don't need to call ensureInitialized
	// here).  If the websocket connection does drop, all jobs started by that
	// connection get stopped automatically.
	return c.sendMessage(req)
}

func (c *Client) registerChannel(name string) *Channel {
	ch := newChannel(c.ctx, name)

	c.Lock()
	c.channelsByName[name] = ch
	c.Unlock()

	return ch
}

func (c *Client) closeRegisteredChannels() {
	c.Lock()
	for _, ch := range c.channelsByName {
		ch.Close()
	}
	c.channelsByName = map[string]*Channel{}
	c.Unlock()
}

// Close the client and shutdown any ongoing connections and goroutines.  The
// client cannot be reused after Close.
func (c *Client) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	c.closeRegisteredChannels()
}
