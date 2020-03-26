package transport

import (
	"encoding/json"
	"github.com/chuckpreslar/emission"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"net"
	"sync"
	"time"
	"github.com/edwsel/ws-proto/logger"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 10 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

const (
	ErrorEvent      = "error"
	MessageEvent    = "message"
	CloseEvent      = "close"
)

type BaseTransport struct {
	emission.Emitter
	ConnectionId string
	socket       *websocket.Conn
	mutex        *sync.Mutex
	closed       bool
}

func NewTransport(socket *websocket.Conn) *BaseTransport {
	var transport BaseTransport

	transport.ConnectionId = uuid.Must(uuid.NewUUID()).String()
	transport.Emitter = *emission.NewEmitter()

	transport.Emitter.RecoverWith(func(event interface{}, listener interface{}, err error) {
		logger.WithError(err).
			WithField("event", event).
			WithField("connection_id", transport.ConnectionId).
			Error("WebSocketServer.BaseTransport.Emitter.RecoverWith")
	})

	transport.socket = socket

	transport.socket.SetCloseHandler(func(code int, message string) error {
		logger.WithField("code", code).
			WithField("message", message).
			WithField("connection_id", transport.ConnectionId).
			Warn("WebSocketServer.BaseTransport.socket.SetCloseHandler")

		transport.Emit("close", code, message)
		transport.Close()

		transport.closed = true
		return nil
	})

	transport.mutex = new(sync.Mutex)
	transport.closed = false

	return &transport
}

func (t *BaseTransport) Read() {
	in := make(chan []byte)
	stop := make(chan struct{})
	pingTicker := time.NewTicker(pingPeriod)

	t.socket.SetReadDeadline(time.Now().Add(pongWait))
	t.socket.SetPongHandler(func(string) error {
		logger.WithField("message", "Send keepalive").
			WithField("connection_id", t.ConnectionId).
			Debug("WebSocketServer.BaseTransport.Read.Pong")

		t.socket.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	var c = t.socket
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logger.WithError(err).
					WithField("connection_id", t.ConnectionId).
					Warn("WebSocketServer.BaseTransport.Read.ReadMessage")

				if c, ok := err.(*websocket.CloseError); ok {
					t.Emit("error", c.Code, c.Text)
				} else {
					if c, k := err.(*net.OpError); k {
						t.Emit("error", websocket.ClosePolicyViolation, c.Error())
					}
				}
				close(stop)
				break
			}
			in <- message
		}
	}()

	for {
		select {
		case _ = <-pingTicker.C:
			logger.WithField("message", "Send keepalive").
				WithField("connection_id", t.ConnectionId).
				Debug("WebSocketServer.BaseTransport.Read.Ping")

			t.socket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := t.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.WithError(NewPingPongError("Ping has field")).
					WithField("connection_id", t.ConnectionId).
					Error("WebSocketServer.BaseTransport.Read.Ping")

				pingTicker.Stop()
				return
			}
		case message := <-in:
			logger.WithField("message", string(message)).
				WithField("connection_id", t.ConnectionId).
				Debug("WebSocketServer.BaseTransport.Read.Received")

			t.Emit("message", []byte(message))
		case <-stop:
			return
		}
	}
}

func (t *BaseTransport) Write(event string, message interface{}) error {
	logger.WithField("event", event).
		WithField("message", message).
		WithField("connection_id", t.ConnectionId).
		Debug("WebSocketServer.BaseTransport.Write")

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return NewWriteClosedError("websocket closed")
	}

	data, err := json.Marshal(NewEvent(event, message))

	if err != nil {
		logger.WithError(err).
			WithField("connection_id", t.ConnectionId).
			Debug("WebSocketServer.BaseTransport.Write.json")
		return err
	}

	logger.WithField("data", string(data)).
		WithField("connection_id", t.ConnectionId).
		Debug("WebSocketServer.BaseTransport.Write.json")

	return t.socket.WriteMessage(websocket.TextMessage, data)
}

func (t *BaseTransport) Close() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.closed == false {
		logger.WithField("connection_id", t.ConnectionId).
			WithField("message", "close ws transport now").
			Info("WebSocketServer.BaseTransport.Close")
		t.socket.Close()
		t.closed = true
	} else {
		logger.WithField("connection_id", t.ConnectionId).
			WithField("message", "transport already closed").
			Warn("WebSocketServer.BaseTransport.Close")
	}
}
