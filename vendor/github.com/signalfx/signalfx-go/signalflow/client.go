package signalflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"path"
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
	conn                   *websocket.Conn
	outgoingMessages       chan interface{}
	readTimeout            time.Duration
	// How long to wait for writes to the websocket to finish
	writeTimeout   time.Duration
	streamURL      *url.URL
	channelsByName map[string]*Channel

	isInitialized bool
	isShutdown    bool
	ctx           context.Context
	cancel        context.CancelFunc
	lastErr       error
	lock          sync.Mutex
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

// NewClient makes a new, but uninitialized, SignalFlow client.
func NewClient(options ...ClientParam) (*Client, error) {
	c := &Client{
		streamURL: &url.URL{
			Scheme: "wss",
			Host:   "stream.us0.signalfx.com",
			Path:   "/v2/signalflow",
		},
		readTimeout:            1 * time.Minute,
		writeTimeout:           5 * time.Second,
		outgoingMessages:       make(chan interface{}),
		channelsByName:         make(map[string]*Channel),
		defaultMetadataTimeout: 5 * time.Second,
	}

	for i := range options {
		if err := options[i](c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) ensureInitialized() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var err error
	if !c.isInitialized {
		err = c.initialize()
		// The mutex also acts as a memory barrier to ensure this write will be
		// seen by any goroutine that obtains the lock after the last Unlock,
		// see https://golang.org/ref/mem#tmp_8.
		c.isInitialized = true
	}
	return err
}

// Assumes c.Mutex is held when called.  Gets the client connection in a state
// that is ready for execute requests.
func (c *Client) initialize() error {
	authenticatedCond := sync.NewCond(&c.lock)

	if c.isShutdown {
		return errors.New("cannot initialize client after shutdown")
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	if c.conn == nil {
		var err error
		c.conn, err = connect(c.ctx, c.streamURL)
		if err != nil {
			return err
		}

		go c.keepWritingMessages()
		go c.keepReadingMessages(authenticatedCond)
		// This just sends off the authenticate request but we have to wait for
		// another websocket message saying that the credentials were valid,
		// after which the authenticatedCond is triggered in the
		// keepReadingMessages loop.
		c.authenticate()
		authenticatedCond.Wait()
	}
	return c.lastErr
}

func (c *Client) newUniqueChannelName() string {
	name := fmt.Sprintf("ch-%d", atomic.AddInt64(&c.nextChannelNum, 1))
	return name
}

// Writes all messages from a single goroutine since that is required by
// websocket library.
func (c *Client) keepWritingMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case message := <-c.outgoingMessages:
			err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err != nil {
				log.Printf("Error setting write timeout for SignalFlow request: %v", err)
				c.lastErr = err
				continue
			}

			msgBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling SignalFlow request: %v", err)
				c.lastErr = err
				continue
			}

			err = c.conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				log.Printf("Error writing SignalFlow request: %v", err)
				c.lastErr = err
				continue
			}
		}
	}
}

// Reads all messages from a single goroutine and distributes them where
// needed.
func (c *Client) keepReadingMessages(authenticatedCond *sync.Cond) {
	for {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			log.Printf("Error setting read timeout in SignalFlow client: %v", err)
			continue
		}
		msgTyp, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			// this means we are shutdown
			if c.ctx.Err() != nil {
				return
			}
			c.lastErr = err
			log.Printf("SignalFlow websocket error: %v", err)
			// This will shut down all computation resources in the client as
			// well.
			c.cancel()
			// The websocket connection is closed by the server if the auth
			// token is bad.
			authenticatedCond.Signal()
			continue
		}

		message, err := messages.ParseMessage(msgBytes, msgTyp == websocket.TextMessage)
		if err != nil {
			log.Printf("Error parsing SignalFlow message: %v", err)
			c.lastErr = err
			continue
		}

		if cm, ok := message.(messages.ChannelMessage); ok {
			channelName := cm.Channel()
			channel, ok := c.channelsByName[channelName]
			if !ok || channelName == "" {
				c.acceptMessage(message, authenticatedCond)
				continue
			}
			channel.AcceptMessage(message)
		} else {
			c.acceptMessage(message, authenticatedCond)
		}
	}
}

// acceptMessages accepts non-channel specific messages.  The only one that I
// know of is the authenticated response.
func (c *Client) acceptMessage(message messages.Message, authenticatedCond *sync.Cond) {
	if _, ok := message.(*messages.AuthenticatedMessage); ok {
		authenticatedCond.Signal()
	} else {
		log.Printf("Unknown SignalFlow message received: %v", message)
	}
}

func connect(ctx context.Context, streamURL *url.URL) (*websocket.Conn, error) {
	connectURL := *streamURL
	connectURL.Path = path.Join(streamURL.Path, "connect")
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, connectURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not connect Signalflow websocket: %v", err)
	}
	return conn, nil
}

// Sends the authenticate message but does not wait for a response.
func (c *Client) authenticate() {
	c.outgoingMessages <- &AuthRequest{
		Token:     c.token,
		UserAgent: c.userAgent,
	}
}

// Execute a SignalFlow job and return a channel upon which informational
// messages and data will flow.
func (c *Client) Execute(req *ExecuteRequest) (*Computation, error) {
	if req.Channel == "" {
		req.Channel = c.newUniqueChannelName()
	}

	if err := c.ensureInitialized(); err != nil {
		return nil, err
	}

	c.outgoingMessages <- req

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
	c.outgoingMessages <- req
	return nil
}

func (c *Client) registerChannel(name string) *Channel {
	ch := newChannel(c.ctx, name)

	c.lock.Lock()
	c.channelsByName[name] = ch
	defer c.lock.Unlock()

	return ch
}

// Close the client and shutdown any ongoing connections and goroutines.
func (c *Client) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	for _, ch := range c.channelsByName {
		ch.Close()
	}
	c.isShutdown = true
}
