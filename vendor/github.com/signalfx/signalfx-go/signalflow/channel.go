package signalflow

import (
	"context"

	"github.com/signalfx/signalfx-go/signalflow/messages"
)

// Channel is a queue of messages that all pertain to the same computation.
type Channel struct {
	name     string
	messages chan messages.Message
	ctx      context.Context
}

func newChannel(ctx context.Context, name string) *Channel {
	c := &Channel{
		name:     name,
		messages: make(chan messages.Message),
		ctx:      ctx,
	}
	return c
}

// AcceptMessage from a websocket.  This might block if nothing is reading from
// the channel but generally a computation should always be doing so.
func (c *Channel) AcceptMessage(msg messages.Message) {
	select {
	case c.messages <- msg:
	case <-c.ctx.Done():
	}
}

// Messages returns a Go chan that will be pushed all of the deserialized
// SignalFlow messages from the websocket.
func (c *Channel) Messages() <-chan messages.Message {
	return c.messages
}

// Close the channel.  This does not actually stop a job in SignalFlow, for
// that use Computation.Stop().
func (c *Channel) Close() {
	close(c.messages)
}
