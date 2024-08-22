package stream

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{}

type StreamHandler struct {
	server *Server
}

func NewStreamHandler(server *Server) *StreamHandler {
	return &StreamHandler{server: server}
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to upgrade endpoint")
		return
	}
	defer c.Close()
	c.SetPongHandler(func(string) error {
		log.Debug().Msg("Received pong from client")
		return nil
	})

	log.Info().Msg("Client has connected to the websocket")
	clientID := uuid.New().String()
	done := make(chan struct{})

	go h.server.PingClient(c, clientID, done)
	h.server.ProcessMessage(c, clientID, done)
}
