package signalflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/signalfx/golib/v3/pointer"
	"github.com/signalfx/signalfx-go/idtool"
	"github.com/signalfx/signalfx-go/signalflow/messages"
)

// Computation is a single running SignalFlow job
type Computation struct {
	ctx     context.Context
	cancel  context.CancelFunc
	channel *Channel
	client  *Client
	dataCh  chan *messages.DataMessage
	// An intermediate channel for data messages where they can be buffered if
	// nothing is currently pulling data messages.
	dataChBuffer       chan *messages.DataMessage
	expirationCh       chan *messages.ExpiredTSIDMessage
	expirationChBuffer chan *messages.ExpiredTSIDMessage
	updateSignal       updateSignal
	lastError          error

	resolutionMS             *int
	lagMS                    *int
	maxDelayMS               *int
	matchedSize              *int
	limitSize                *int
	matchedNoTimeseriesQuery *string
	groupByMissingProperties []string

	tsidMetadata map[idtool.ID]*messages.MetadataProperties
	events       []*messages.EventMessage

	handle string

	// The timeout to wait for metadata when a metadata access function is
	// called.  This will default to what is set on the client, but can be
	// overridden by changing this field directly.
	MetadataTimeout time.Duration
}

// ComputationError exposes the underlying metadata of a computation error
type ComputationError struct {
	Code      int
	Message   string
	ErrorType string
}

func (e *ComputationError) Error() string {
	err := fmt.Sprintf("%v", e.Code)
	if e.ErrorType != "" {
		err = fmt.Sprintf("%v (%v)", e.Code, e.ErrorType)
	}
	if e.Message != "" {
		err = fmt.Sprintf("%v: %v", err, e.Message)
	}
	return err
}

func newComputation(ctx context.Context, channel *Channel, client *Client) *Computation {
	newCtx, cancel := context.WithCancel(ctx)
	comp := &Computation{
		ctx:                newCtx,
		cancel:             cancel,
		channel:            channel,
		client:             client,
		dataCh:             make(chan *messages.DataMessage),
		dataChBuffer:       make(chan *messages.DataMessage),
		expirationCh:       make(chan *messages.ExpiredTSIDMessage),
		expirationChBuffer: make(chan *messages.ExpiredTSIDMessage),
		tsidMetadata:       make(map[idtool.ID]*messages.MetadataProperties),
		updateSignal:       updateSignal{},
		MetadataTimeout:    client.defaultMetadataTimeout,
	}

	go comp.bufferDataMessages()
	go comp.bufferExpirationMessages()
	go comp.watchMessages()
	return comp
}

// Channel returns the underlying Channel instance used by this computation.
func (c *Computation) Channel() *Channel {
	return c.channel
}

// Handle of the computation
func (c *Computation) Handle() string {
	if err := c.waitForMetadata(func() bool { return c.handle != "" }); err != nil {
		return ""
	}
	return c.handle
}

// Waits for the given cond func to return true, or until the metadata timeout
// duration has passed.
func (c *Computation) waitForMetadata(cond func() bool) error {
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	remaining := c.MetadataTimeout
	for !cond() {
		if err := c.updateSignal.WaitWithTimeout(c.ctx, &remaining); err != nil {
			return err
		}
	}
	return nil
}

// Resolution of the job.  This will wait for a short while for the resolution
// message to come on the websocket, but will return 0 after a timeout if it
// does not come.
func (c *Computation) Resolution() time.Duration {
	if err := c.waitForMetadata(func() bool { return c.resolutionMS != nil }); err != nil {
		return 0
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return time.Duration(*c.resolutionMS) * time.Millisecond
}

// Lag detected for the job.  This will wait for a short while for the lag
// message to come on the websocket, but will return 0 after a timeout if it
// does not come.
func (c *Computation) Lag() time.Duration {
	if err := c.waitForMetadata(func() bool { return c.lagMS != nil }); err != nil {
		return 0
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return time.Duration(*c.lagMS) * time.Millisecond
}

// MaxDelay detected of the job.  This will wait for a short while for the max
// delay message to come on the websocket, but will return 0 after a timeout if
// it does not come.
func (c *Computation) MaxDelay() time.Duration {
	if err := c.waitForMetadata(func() bool { return c.maxDelayMS != nil }); err != nil {
		return 0
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return time.Duration(*c.maxDelayMS) * time.Millisecond
}

// MatchedSize detected of the job.  This will wait for a short while for the matched
// size message to come on the websocket, but will return 0 after a timeout if
// it does not come.
func (c *Computation) MatchedSize() int {
	if err := c.waitForMetadata(func() bool { return c.matchedSize != nil }); err != nil {
		return 0
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return *c.matchedSize
}

// LimitSize detected of the job.  This will wait for a short while for the limit
// size message to come on the websocket, but will return 0 after a timeout if
// it does not come.
func (c *Computation) LimitSize() int {
	if err := c.waitForMetadata(func() bool { return c.limitSize != nil }); err != nil {
		return 0
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return *c.limitSize
}

// MatchedNoTimeseriesQuery if it matched no active timeseries.
// This will wait for a short while for the limit
// size message to come on the websocket, but will return "" after a timeout if
// it does not come.
func (c *Computation) MatchedNoTimeseriesQuery() string {
	if err := c.waitForMetadata(func() bool { return c.matchedNoTimeseriesQuery != nil }); err != nil {
		return ""
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return *c.matchedNoTimeseriesQuery
}

// GroupByMissingProperties are timeseries that don't contain the required dimensions.
// This will wait for a short while for the limit
// size message to come on the websocket, but will return nil after a timeout if
// it does not come.
func (c *Computation) GroupByMissingProperties() []string {
	if err := c.waitForMetadata(func() bool { return c.groupByMissingProperties != nil }); err != nil {
		return nil
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return c.groupByMissingProperties
}

// TSIDMetadata for a particular tsid.  This will wait for a short while for
// the tsid metadata message to come on the websocket, but will return nil
// after a timeout if it does not come.
func (c *Computation) TSIDMetadata(tsid idtool.ID) *messages.MetadataProperties {
	if err := c.waitForMetadata(func() bool { return c.tsidMetadata[tsid] != nil }); err != nil {
		return nil
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return c.tsidMetadata[tsid]
}

// Events returns the results from events or alerts queries.
func (c *Computation) Events() []*messages.EventMessage {
	if err := c.waitForMetadata(func() bool { return c.events != nil }); err != nil {
		return nil
	}
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()
	return c.events
}

// Done passes through the computation context's Done channel for use in select
// statements to know when the computation is finished or an error occurred.
func (c *Computation) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the last fatal error that caused the computation to stop, if
// any.  Will be nil if the computation stopped in an expected manner.
func (c *Computation) Err() error {
	return c.lastError
}

func (c *Computation) watchMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case m, ok := <-c.channel.Messages():
			if !ok {
				c.cancel()
				continue
			}
			c.processMessage(m)
		}
	}
}

func (c *Computation) processMessage(m messages.Message) {
	defer c.updateSignal.SignalAll()
	c.updateSignal.Lock()
	defer c.updateSignal.Unlock()

	switch v := m.(type) {
	case *messages.JobStartControlMessage:
		c.handle = v.Handle
	case *messages.EndOfChannelControlMessage, *messages.ChannelAbortControlMessage:
		c.cancel()
	case *messages.DataMessage:
		c.dataChBuffer <- v
	case *messages.ExpiredTSIDMessage:
		delete(c.tsidMetadata, idtool.IDFromString(v.TSID))
		c.expirationChBuffer <- v
	case *messages.InfoMessage:
		switch v.MessageBlock.Code {
		case messages.JobRunningResolution:
			c.resolutionMS = pointer.Int(v.MessageBlock.Contents.(messages.JobRunningResolutionContents).ResolutionMS())
		case messages.JobDetectedLag:
			c.lagMS = pointer.Int(v.MessageBlock.Contents.(messages.JobDetectedLagContents).LagMS())
		case messages.JobInitialMaxDelay:
			c.maxDelayMS = pointer.Int(v.MessageBlock.Contents.(messages.JobInitialMaxDelayContents).MaxDelayMS())
		case messages.FindLimitedResultSet:
			c.matchedSize = pointer.Int(v.MessageBlock.Contents.(messages.FindLimitedResultSetContents).MatchedSize())
			c.limitSize = pointer.Int(v.MessageBlock.Contents.(messages.FindLimitedResultSetContents).LimitSize())
		case messages.FindMatchedNoTimeseries:
			c.matchedNoTimeseriesQuery = pointer.String(v.MessageBlock.Contents.(messages.FindMatchedNoTimeseriesContents).MatchedNoTimeseriesQuery())
		case messages.GroupByMissingProperty:
			c.groupByMissingProperties = v.MessageBlock.Contents.(messages.GroupByMissingPropertyContents).GroupByMissingProperties()
		}
	case *messages.ErrorMessage:
		rawData := v.RawData()
		computationError := ComputationError{}
		if code, ok := rawData["error"]; ok {
			computationError.Code = int(code.(float64))
		}
		if msg, ok := rawData["message"]; ok && msg != nil {
			computationError.Message = msg.(string)
		}
		if errType, ok := rawData["errorType"]; ok {
			computationError.ErrorType = errType.(string)
		}
		c.lastError = &computationError
		c.cancel()
	case *messages.MetadataMessage:
		c.tsidMetadata[v.TSID] = &v.Properties
	case *messages.EventMessage:
		c.events = append(c.events, v)
	}
}

// Buffer up data messages indefinitely until another goroutine reads them off of
// c.messages, which is an unbuffered channel.
func (c *Computation) bufferDataMessages() {
	buffer := make([]*messages.DataMessage, 0)
	var nextMessage *messages.DataMessage
	for {
		if len(buffer) > 0 {
			if nextMessage == nil {
				nextMessage, buffer = buffer[0], buffer[1:]
			}
			select {
			case <-c.ctx.Done():
				close(c.dataCh)
				return
			case c.dataCh <- nextMessage:
				nextMessage = nil
			case msg := <-c.dataChBuffer:
				buffer = append(buffer, msg)
			}
		} else {
			select {
			case <-c.ctx.Done():
				close(c.dataCh)
				return
			case msg := <-c.dataChBuffer:
				buffer = append(buffer, msg)
			}
		}
	}
}

// Buffer up expiration messages indefinitely until another goroutine reads
// them off of c.expirationCh, which is an unbuffered channel.
func (c *Computation) bufferExpirationMessages() {
	buffer := make([]*messages.ExpiredTSIDMessage, 0)
	var nextMessage *messages.ExpiredTSIDMessage
	for {
		if len(buffer) > 0 {
			if nextMessage == nil {
				nextMessage, buffer = buffer[0], buffer[1:]
			}

			select {
			case <-c.ctx.Done():
				return
			case c.expirationCh <- nextMessage:
				nextMessage = nil
			case msg := <-c.expirationChBuffer:
				buffer = append(buffer, msg)
			}
		} else {
			select {
			case <-c.ctx.Done():
				return
			case msg := <-c.expirationChBuffer:
				buffer = append(buffer, msg)
			}
		}
	}
}

// Data returns the channel on which data messages come.
func (c *Computation) Data() <-chan *messages.DataMessage {
	return c.dataCh
}

// Expirations returns a channel that will be sent messages about expired
// TSIDs, i.e. time series that are no longer valid for this computation.
func (c *Computation) Expirations() <-chan *messages.ExpiredTSIDMessage {
	return c.expirationCh
}

// IsFinished returns true if the computation is done and no more data should
// be expected from it.
func (c *Computation) IsFinished() bool {
	// The context will have a non-nil err if it was cancelled.
	return c.ctx.Err() != nil
}

// Stop the computation on the backend.
func (c *Computation) Stop() error {
	return c.StopWithReason("")
}

// StopWithReason stops the computation with a given reason. This reason will
// be reflected in the control message that signals the end of the job/channel.
func (c *Computation) StopWithReason(reason string) error {
	return c.client.Stop(&StopRequest{
		Reason: reason,
		Handle: c.handle,
	})
}

// Simple struct that allows one goroutine to signal a bunch of other
// goroutines that are waiting on a condition, with a timeout.  It is basically
// similar to sync.Cond except the lock in internal (but accessible) and you
// can set a timeout.
type updateSignal struct {
	sync.Mutex
	s chan struct{}
}

// WaitWithTimeout waits for the given duration, remaining, for the signal to
// be triggered. It is assumed that the caller holds u.Mutex upon calling.
// When this function returns, that mutex will be relocked, but will have been
// unlocked for some time while waiting.  The remaining arg will be updated in
// place to contain the remaining time when the function returned.
func (u *updateSignal) WaitWithTimeout(ctx context.Context, remaining *time.Duration) error {
	start := time.Now()

	if u.s == nil {
		u.reset()
	}
	sig := u.s
	u.Unlock()
	defer func() {
		newRemaining := *remaining - time.Since(start)
		if newRemaining > 0 {
			*remaining = newRemaining
		} else {
			*remaining = 0
		}
	}()
	defer u.Lock()

	ctxTimeout, cancel := context.WithTimeout(ctx, *remaining)
	defer cancel()
	select {
	case <-ctxTimeout.Done():
		return ctxTimeout.Err()
	case <-sig:
		return nil
	}
}

func (u *updateSignal) reset() {
	u.s = make(chan struct{})
}

func (u *updateSignal) SignalAll() {
	u.Lock()
	if u.s != nil {
		close(u.s)
		u.reset()
	}
	u.Unlock()
}
