package peer

import (
	"context"
	"github.com/chuckpreslar/emission"
	"github.com/google/uuid"
	"ws-proto/transport"
)

const (
	EmptyEvent   = "empty"
	ErrorEvent   = "error"
	MessageEvent = "message"
	CloseEvent   = "close"
)

type Peer struct {
	*emission.Emitter
	uid        uuid.UUID
	connection *transport.BaseTransport
	ctx        context.Context
}

func New(ctx context.Context, connection *transport.BaseTransport) *Peer {
	return &Peer{
		Emitter:    emission.NewEmitter(),
		connection: connection,
		ctx:        ctx,
	}
}

func (p *Peer) Uid() string {
	return p.connection.ConnectionId
}

func (p *Peer) Context() context.Context {
	return p.ctx
}

func (p *Peer) Send(event string, message interface{}) error {
	return p.connection.Write(event, message)
}

func (p *Peer) Close() {
	p.connection.Close()
}
