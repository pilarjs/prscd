package psig

import (
	"github.com/vmihailenco/msgpack/v5"
	"github.com/yomorun/yomo/serverless"
)

// PrscdDataTag is the tag of prscd data.
const PrscdDataTag uint32 = 0x21

// NewContext returns context for prscd from serverless.Context.
// The Context can be used to load events and write events.
func NewContext(ctx serverless.Context, sfnName string) (PrscdContext, error) {
	var signal Signalling
	if err := msgpack.Unmarshal(ctx.Data(), &signal); err != nil {
		return nil, err
	}

	pCtx := &prscdContext{
		ctx:            ctx,
		baseSignalling: &signal,
		name:           sfnName,
	}
	return pCtx, nil
}

// PrscdContext is the context for prscd,
// you can load events from ctx and write events to it.
type PrscdContext interface {
	// ReadEvent loads event from ctx.
	ReadEvent() (*ChannelEvent, error)
	// WriteEvent writes event to ctx.
	WriteEvent(event *ChannelEvent) error
	// ReadSignalling returns the Signalling.
	ReadSignalling() *Signalling
}

type prscdContext struct {
	ctx            serverless.Context
	baseSignalling *Signalling
	name           string
}

func (c *prscdContext) ReadSignalling() *Signalling {
	return c.baseSignalling
}

func (c *prscdContext) ReadEvent() (*ChannelEvent, error) {
	payload := c.baseSignalling.Payload

	var ev ChannelEvent
	if err := msgpack.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}

	return &ev, nil
}

func (c *prscdContext) WriteEvent(event *ChannelEvent) error {
	signallingPayload, err := msgpack.Marshal(&event)
	if err != nil {
		return err
	}
	c.baseSignalling.Payload = signallingPayload

	// when writing, set Sid and Cid as the name of the serverless function
	c.baseSignalling.Sid = c.name
	c.baseSignalling.Cid = c.name

	buf, err := msgpack.Marshal(&c.baseSignalling)
	if err != nil {
		return err
	}

	return c.ctx.Write(PrscdDataTag, buf)
}
