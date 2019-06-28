package signalflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/signalfx/golib/pointer"
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
	dataChBuffer chan *messages.DataMessage
	updateSignal updateSignal
	lastError    error

	resolutionMS *int
	lagMS        *int
	maxDelayMS   *int

	tsidMetadata map[idtool.ID]*messages.MetadataProperties

	handle string

	// The timeout to wait for metadata when a metadata access function is
	// called.  This will default to what is set on the client, but can be
	// overridden by changing this field directly.
	MetadataTimeout time.Duration
}

func newComputation(ctx context.Context, channel *Channel, client *Client) *Computation {
	newCtx, cancel := context.WithCancel(ctx)
	comp := &Computation{
		ctx:             newCtx,
		cancel:          cancel,
		channel:         channel,
		client:          client,
		dataCh:          make(chan *messages.DataMessage),
		dataChBuffer:    make(chan *messages.DataMessage),
		tsidMetadata:    make(map[idtool.ID]*messages.MetadataProperties),
		updateSignal:    updateSignal{},
		MetadataTimeout: client.defaultMetadataTimeout,
	}

	go comp.bufferDataMessages()
	go comp.watchMessages()
	return comp
}

// Channel returns the underlying Channel instance used by this computation.
func (c *Computation) Channel() *Channel {
	return c.channel
}

// Handle of the computation
func (c *Computation) Handle() string {
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
	return time.Duration(*c.resolutionMS) * time.Millisecond
}

// Lag detected for the job.  This will wait for a short while for the lag
// message to come on the websocket, but will return 0 after a timeout if it
// does not come.
func (c *Computation) Lag() time.Duration {
	if err := c.waitForMetadata(func() bool { return c.lagMS != nil }); err != nil {
		return 0
	}
	return time.Duration(*c.lagMS) * time.Millisecond
}

// MaxDelay detected of the job.  This will wait for a short while for the max
// delay message to come on the websocket, but will return 0 after a timeout if
// it does not come.
func (c *Computation) MaxDelay() time.Duration {
	if err := c.waitForMetadata(func() bool { return c.maxDelayMS != nil }); err != nil {
		return 0
	}
	return time.Duration(*c.maxDelayMS) * time.Millisecond
}

// TSIDMetadata for a particular tsid.  This will wait for a short while for
// the tsid metadata message to come on the websocket, but will return nil
// after a timeout if it does not come.
func (c *Computation) TSIDMetadata(tsid idtool.ID) *messages.MetadataProperties {
	if err := c.waitForMetadata(func() bool { return c.tsidMetadata[tsid] != nil }); err != nil {
		return nil
	}
	return c.tsidMetadata[tsid]
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
			close(c.dataCh)
			return
		case m := <-c.channel.Messages():
			c.processMessage(m)
		}
	}
}

func (c *Computation) processMessage(m messages.Message) {
	defer c.updateSignal.SignalAll()

	switch v := m.(type) {
	case *messages.JobStartControlMessage:
		c.handle = v.Handle
	case *messages.BaseControlMessage:
		switch v.Type() {
		case messages.ChannelAbortEvent, messages.EndOfChannelEvent:
			c.cancel()
		}
	case *messages.DataMessage:
		c.dataChBuffer <- v
	case *messages.InfoMessage:
		switch v.MessageBlock.Code {
		case messages.JobRunningResolution:
			c.resolutionMS = pointer.Int(v.MessageBlock.Contents.(messages.JobRunningResolutionContents).ResolutionMS())
		case messages.JobDetectedLag:
			c.lagMS = pointer.Int(v.MessageBlock.Contents.(messages.JobDetectedLagContents).LagMS())
		case messages.JobInitialMaxDelay:
			c.maxDelayMS = pointer.Int(v.MessageBlock.Contents.(messages.JobInitialMaxDelayContents).MaxDelayMS())
		}
	case *messages.ErrorMessage:
		c.lastError = fmt.Errorf("error executing SignalFlow: %v", v.RawData())
		c.cancel()
	case *messages.MetadataMessage:
		c.tsidMetadata[v.TSID] = &v.Properties
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
				return
			case c.dataCh <- nextMessage:
				nextMessage = nil
			case msg := <-c.dataChBuffer:
				buffer = append(buffer, msg)
			}
		} else {
			buffer = append(buffer, <-c.dataChBuffer)
		}
	}
}

// Data returns the channel on which data messages come.
func (c *Computation) Data() <-chan *messages.DataMessage {
	return c.dataCh
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
