package connection

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

type Connection struct {
	*emission.Emitter
	uid        uuid.UUID
	connection *transport.BaseTransport
	ctx        context.Context
}

func New(ctx context.Context, connection *transport.BaseTransport) *Connection {
	emitter := emission.NewEmitter()
	emitter.RecoverWith(func(event interface{}, i2 interface{}, err error) {
		_, f, l, _ := runtime.Caller(1)

		logger.WithError(err).
			WithField("event", event).
			WithField("connection_id", connection.ConnectionId).
			WithField("file", f).
			WithField("line", l).
			WithField("stack", string(debug.Stack())).
			Error("WebSocketServer.connection.RecoverWith")
	})

	return &Connection{
		Emitter:    emitter,
		connection: connection,
		ctx:        ctx,
	}
}

func (p *Connection) Uid() string {
	return p.connection.ConnectionId
}

func (p *Connection) Context() context.Context {
	return p.ctx
}

func (p *Connection) Send(event string, message interface{}) error {
	return p.connection.Write(event, message)
}

func (p *Connection) Close() {
	p.connection.Close()
}
