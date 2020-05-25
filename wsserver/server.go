package wsserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chuckpreslar/emission"
	"github.com/edwsel/ws-proto/handlers"
	"github.com/edwsel/ws-proto/logger"
	"github.com/edwsel/ws-proto/connection"
	"github.com/edwsel/ws-proto/transport"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Handel func(ws *transport.BaseTransport, request *http.Request)
type BeforeUpgrader func(request *http.Request, fail RaiseFail) context.Context
type OnConnection func(peer *connection.Connection, request *http.Request)
type RaiseFail func(message string, code int, data ...interface{})

func DefaultUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   0,
		WriteBufferSize:  0,
		WriteBufferPool:  nil,
		Subprotocols: []string{
			"protoo",
		},
		Error: nil,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
	}
}

func DefaultBeforeUpgrader(request *http.Request, fail RaiseFail) context.Context {
	return context.TODO()
}

type WebsocketServer struct {
	*emission.Emitter
	upgrader       websocket.Upgrader
	beforeUpgrader BeforeUpgrader
	handler        *handlers.Handler
	path           string
}

func New() *WebsocketServer {
	return &WebsocketServer{
		Emitter:        emission.NewEmitter(),
		upgrader:       DefaultUpgrader(),
		beforeUpgrader: DefaultBeforeUpgrader,
		handler:        handlers.New(),
		path:           "/stream",
	}
}

func (server *WebsocketServer) SetLogLevel(level logrus.Level) {
	logger.SetLevel(level)
}

func (server *WebsocketServer) SetLogFormat(format logrus.Formatter)  {
	logger.SetFormatter(format)
}

func (server *WebsocketServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.WithField("error", rec).
				Error("WebSocketServer.ServeHTTP")
		}
	}()

	fail := getRaiseFail(writer, request)

	ctx, err := server.runBeforeUpgrader(request, fail)

	if err != nil {
		switch err.(type) {
		case *RaiseFailUsageError:
			logger.WithError(err).
				Error("WebSocketServer.ServeHTTP.RaiseFail")
		case error:
			logger.WithError(err).
				Error("WebSocketServer.ServeHTTP.beforeUpgrader")

			responseError(writer, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	conn, err := server.upgrader.Upgrade(writer, request, http.Header{})

	if err != nil {
		logger.WithError(err).
			Error("WebSocketServer.ServeHTTP.upgrader")

		responseError(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	wsTransport := transport.NewTransport(conn)

	server.handler.Processing(ctx, wsTransport, request)

	wsTransport.Read()
}

func (server *WebsocketServer) SetBeforeUpgrader(upgrader BeforeUpgrader) {
	server.beforeUpgrader = upgrader

	logger.WithField("message", "set new before upgrader").
		Debug("WebSocketServer.SetUpgrader")
}

func (server *WebsocketServer) SetUpgrader(upgrader websocket.Upgrader) {
	server.upgrader = upgrader

	logger.WithField("message", "set new upgrader").
		Debug("WebSocketServer.SetUpgrader")
}

func (server *WebsocketServer) SetPath(path string) {
	server.path = path

	logger.WithField("message", "set new path").
		Debug("WebSocketServer.SetPath")
}

func (server *WebsocketServer) OnConnection(handler OnConnection) {
	server.handler.Emitter.On(handlers.ConnectionEvent, handler)
}

func (server *WebsocketServer) ListenAndServe(addr string) {
	http.Handle(server.path, server)

	logger.WithField("message", "Server start").
		WithField("addr", addr).
		WithField("path", server.path).
		Info("WebSocketServer.ListenAndServe")

	err := http.ListenAndServe(addr, nil)

	logger.WithError(err).
		Panic("WebSocketServer.ListenAndServe")
}

func (server *WebsocketServer) ListenAndServeTLS(addr string, cert string, key string) {
	http.Handle(server.path, server)

	logger.WithField("message", "Server start").
		WithField("addr", addr).
		WithField("path", server.path).
		Info("WebSocketServer.ListenAndServeTLS")

	err := http.ListenAndServeTLS(addr, cert, key, nil)

	logger.WithError(err).
		Panic("WebSocketServer.ListenAndServeTLS")
}

func (server *WebsocketServer) runBeforeUpgrader(request *http.Request, fail RaiseFail) (ctx context.Context, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			switch rec.(type) {
			case error:
				err = rec.(error)
			default:
				err = errors.New(fmt.Sprintf("undefinded error: %v", rec))
			}
		}
	}()

	return server.beforeUpgrader(request, fail), nil
}

func getRaiseFail(writer http.ResponseWriter, request *http.Request) RaiseFail {
	return func(message string, code int, data ...interface{}) {
		response, err := json.Marshal(map[string]interface{}{
			"error": message,
			"data":  data,
		})

		if err != nil {
			code = http.StatusInternalServerError
			logger.Errorf("WebSocketServer.riseFail.json: %v", err)
		}

		writer.WriteHeader(code)

		_, err = writer.Write(response)

		if err != nil {
			logger.WithError(err).
				Error("WebSocketServer.riseFail.write")
		}

		panic(NewRaiseFailUsageError(message))
	}
}

func responseError(writer http.ResponseWriter, message string, code int, data ...interface{}) {
	response, err := json.Marshal(map[string]interface{}{
		"error": message,
		"data":  data,
	})

	if err != nil {
		code = http.StatusInternalServerError
		logger.Errorf("WebSocketServer.responseError.json: %v", err)
	}

	writer.WriteHeader(code)

	_, err = writer.Write(response)

	if err != nil {
		logger.WithError(err).
			Error("WebSocketServer.responseError.write")
	}
}
