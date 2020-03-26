package handlers

import (
	"context"
	"encoding/json"
	"github.com/chuckpreslar/emission"
	"github.com/edwsel/ws-proto/logger"
	"github.com/edwsel/ws-proto/peer"
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

func (h *Handler) Processing(ctx context.Context, connection *transport.BaseTransport, request *http.Request) {
	peerConnection := peer.New(ctx, connection)

	h.Emitter.RecoverWith(func(event interface{}, listener interface{}, err error) {
		_, _, l, _ := runtime.Caller(1)

		logger.WithError(err).
			WithField("event", event).
			WithField("connection_id", connection.ConnectionId).
			WithField("line", l).
			WithField("stack", string(debug.Stack())).
			Error("WebSocketServer.Handler.emitterRecovery")

		peerConnection.Close()
	})

	logger.WithField("message", "Peer connected").
		WithField("connection_id", peerConnection.Uid()).
		Info("WebSocketServer.Handler.Processing.Peer")

	connection.On(transport.ErrorEvent, func(code int, message string) {
		peerConnection.Emit(peer.ErrorEvent, peerConnection, code, message)
	})

	connection.On(transport.CloseEvent, func(code int, message string) {
		peerConnection.Emit(peer.CloseEvent, peerConnection, code, message)
	})

	connection.On(transport.MessageEvent, func(message []byte) {
		h.eventProcessing(peerConnection, message)
	})

	h.Emit(ConnectionEvent, peerConnection, request)
}

func (h *Handler) eventProcessing(currentPeer *peer.Peer, data []byte) {
	message, err := parseMessage(data)

	switch err.(type) {
	case *EmptyEventError:
		logger.WithField("message", err.Error()).
			WithField("connection_id", currentPeer.Uid()).
			Warn("WebSocketServer.Handler.eventProcessing.parseMessage")

		currentPeer.Emit(peer.EmptyEvent, currentPeer, data)

		return
	case error:
		logger.WithError(err).
			WithField("connection_id", currentPeer.Uid()).
			Error("WebSocketServer.Handler.eventProcessing")

		return
	}

	currentPeer.Emit(peer.MessageEvent, currentPeer, message.Event, message.Data)

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
