package handlers

import (
	"context"
	"encoding/json"
	"github.com/chuckpreslar/emission"
	"github.com/edwsel/ws-proto/connection"
	"github.com/edwsel/ws-proto/logger"
	"github.com/edwsel/ws-proto/transport"
	"net/http"
	"runtime"
	"runtime/debug"
)

const (
	ConnectionEvent = "connection"
)

type Message struct {
	Event string      `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

type Handler struct {
	*emission.Emitter
}

func New() *Handler {
	return &Handler{
		Emitter: emission.NewEmitter(),
	}
}

func (h *Handler) Processing(ctx context.Context, transport *transport.BaseTransport, request *http.Request) {
	peerConnection := connection.New(ctx, transport)

	h.Emitter.RecoverWith(func(event interface{}, listener interface{}, err error) {
		_, f, l, _ := runtime.Caller(1)

		logger.WithError(err).
			WithField("event", event).
			WithField("connection_id", transport.ConnectionId).
			WithField("file", f).
			WithField("line", l).
			WithField("stack", string(debug.Stack())).
			Error("WebSocketServer.Handler.emitterRecovery")

		peerConnection.Close()
	})

	logger.WithField("message", "Connection connected").
		WithField("connection_id", peerConnection.Uid()).
		Info("WebSocketServer.Handler.Processing.Connection")

	transport.On(connection.ErrorEvent, func(code int, message string) {
		peerConnection.Emit(connection.ErrorEvent, peerConnection, code, message)
	})

	transport.On(connection.CloseEvent, func(code int, message string) {
		peerConnection.Emit(connection.CloseEvent, peerConnection, code, message)
	})

	transport.On(connection.MessageEvent, func(message []byte) {
		h.eventProcessing(peerConnection, message)
	})

	h.Emit(ConnectionEvent, peerConnection, request)
}

func (h *Handler) eventProcessing(currentPeer *connection.Connection, data []byte) {
	message, err := parseMessage(data)

	switch err.(type) {
	case *EmptyEventError:
		logger.WithField("message", err.Error()).
			WithField("connection_id", currentPeer.Uid()).
			Warn("WebSocketServer.Handler.eventProcessing.parseMessage")

		currentPeer.Emit(connection.EmptyEvent, currentPeer, data)

		return
	case error:
		logger.WithError(err).
			WithField("connection_id", currentPeer.Uid()).
			Error("WebSocketServer.Handler.eventProcessing")

		return
	}

	currentPeer.Emit(connection.MessageEvent, currentPeer, message.Event, message.Data)

	currentPeer.Emit(message.Event, currentPeer, message.Data)
}

func parseMessage(data []byte) (*Message, error) {
	message := new(Message)

	err := json.Unmarshal(data, &message)

	if err != nil {
		return nil, err
	}

	if message.Event == "" {
		return nil, NewEmptyEventError()
	}

	return message, nil
}
