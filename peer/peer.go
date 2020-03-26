package peer

import (
	"context"
	"github.com/chuckpreslar/emission"
	"github.com/edwsel/ws-proto/logger"
	"github.com/edwsel/ws-proto/transport"
	"github.com/google/uuid"
	"runtime"
	"runtime/debug"
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
	emitter := emission.NewEmitter()
	emitter.RecoverWith(func(i interface{}, i2 interface{}, err error) {
		_, _, l, _ := runtime.Caller(1)

		logger.WithError(err).
			WithField("line", l).
			WithField("stack", string(debug.Stack())).
			Error("WebSocketServer.peer.RecoverWith")
	})

	return &Peer{
		Emitter:    emitter,
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
