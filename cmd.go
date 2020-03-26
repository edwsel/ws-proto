package main

import (
	"context"
	"github.com/edwsel/ws-proto/logger"
	"github.com/edwsel/ws-proto/connection"
	"github.com/edwsel/ws-proto/wsserver"
	"net/http"
	"sync"
)

var room sync.Map

func main() {
	server := wsserver.New()
	server.SetBeforeUpgrader(func(request *http.Request, fail wsserver.RaiseFail) context.Context {
		vars := request.URL.Query()
		peerId := vars["connection"][0]

		ctx := context.TODO()
		return context.WithValue(ctx, "peer_id", peerId)
	})
	server.OnConnection(func(currentPeer *connection.Connection, request *http.Request) {
		room.Store(currentPeer.Uid(), currentPeer)

		currentPeer.On(connection.MessageEvent, func(c *connection.Connection, method string, data interface{}) {
			logger.Debug(method, data)
		})

		currentPeer.On("sdp", func(data interface{}) {
			room.Range(func(key, value interface{}) bool {
				subPeer := value.(*connection.Connection)

				if key != currentPeer.Uid() {
					subPeer.Send("sdp", data)
				}
				return true
			})
		})

		currentPeer.Send("connection", map[string]interface{}{
			"uid":     currentPeer.Uid(),
			"peer_id": currentPeer.Context().Value("peer_id"),
		})

		currentPeer.On(connection.CloseEvent, func(connection *connection.Connection, code int, message string) {
			logger.Debug("dd", code, message)
		})
	})

	server.ListenAndServe(":9999")
}
