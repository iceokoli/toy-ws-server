package stream

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var (
	SUBSCRIBE   string = "SUBSCRIBE"
	UNSUBSCRIBE string = "UNSUBSCRIBE"
)

type (
	Clients       map[string]*websocket.Conn
	Subscriptions map[string]Clients
	Tick          struct {
		S string  `json:"s"`
		P float64 `json:"p"`
	}

	Request struct {
		Method  string   `json:"method"`
		Streams []string `json:"streams"`
		Id      string   `json:"id"`
	}

	Result struct {
		Result *string `json:"result"`
		Id     string  `json:"id"`
	}
)

type MarketData interface {
	GetPrice(string) (float64, error)
}

// might need a mutex
type SubscriptionManager struct {
	subscriptions Subscriptions
}

type Server struct {
	manager    SubscriptionManager
	marketData MarketData
	pingFreq   time.Duration
	pongWait   time.Duration
}

func NewServer(md MarketData, streams []string, pingFreq time.Duration, pongWait time.Duration) *Server {
	subs := make(Subscriptions)
	for _, stream := range streams {
		subs[stream] = make(Clients)
	}
	manager := SubscriptionManager{subscriptions: subs}

	return &Server{manager: manager, marketData: md, pingFreq: pingFreq, pongWait: pongWait}
}

func (s *SubscriptionManager) AddClient(clientID string, streams []string, c *websocket.Conn) error {
	for _, stream := range streams {
		s, ok := s.subscriptions[stream]
		if !ok {
			return fmt.Errorf("stream %s does not exist", stream)
		}
		s[clientID] = c
	}
	return nil
}

func (s *SubscriptionManager) RemoveClient(clientID string, streams []string) error {
	if streams == nil {
		for _, subscriptions := range s.subscriptions {
			delete(subscriptions, clientID)
		}
	}
	for _, stream := range streams {
		s, ok := s.subscriptions[stream]
		if !ok {
			return fmt.Errorf("stream %s does not exist", stream)
		}
		delete(s, clientID)
	}
	return nil
}

func (s SubscriptionManager) GetSubscriptions() Subscriptions {
	return s.subscriptions
}

func (s *Server) ProcessMessage(c *websocket.Conn, clientID string, done chan struct{}) {
	defer func() {
		close(done)
	}()

	for {
		_, m, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				log.Info().Msg("Connection was closed by the client, removing from the subscription list")
			} else {
				log.Error().Stack().Err(err).Msg("error occured while reading a message, closing connection")
			}
			s.manager.RemoveClient(clientID, nil)
			return
		}

		req := Request{}
		if err := json.Unmarshal(m, &req); err != nil {
			log.Error().Stack().Err(err).Msg("bad request received from the client, closing connection")
			return
		}

		log.Debug().Str("method", req.Method).Str("client_id", clientID).Msgf("Got message from client streams = %v", req.Streams)
		switch req.Method {
		case SUBSCRIBE:
			if err := s.subscribe(c, clientID, req.Streams); err != nil {
				log.Error().Stack().Err(err).Msgf("Failed to subscribe client %v, closing connection", clientID)
				s.manager.RemoveClient(clientID, nil)
				return
			}
		case UNSUBSCRIBE:
			if err := s.unSubscribe(c, clientID, req.Streams); err != nil {
				log.Error().Stack().Err(err).Msgf("Failed to unsubscribe client %v, closing connection", clientID)
				s.manager.RemoveClient(clientID, nil)
				return
			}
		default:
			log.Info().Str("method", req.Method).Msg("Unsupported method, closing connection")
			s.manager.RemoveClient(clientID, nil)
			return
		}
	}
}

func (s *Server) PingClient(c *websocket.Conn, clientID string, done <-chan struct{}) {
	ticker := time.NewTicker(s.pingFreq)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Debug().Str("client_id", clientID).Msg("Sending ping message to client")
			if err := c.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(s.pongWait)); err != nil {
				log.Error().Stack().Err(err).Msg("failed to send ping message, removing cient subscription")
				s.manager.RemoveClient(clientID, nil)
				return
			}
		case <-done:
			return
		}
	}
}

func (s *Server) PublishPrices(freq time.Duration) {
	ticker := time.NewTicker(freq)
	defer ticker.Stop()

	log.Info().Msg("Ready to publish prices to subscribers")

	for {
		select {
		case <-ticker.C:
			for stream, clients := range s.manager.GetSubscriptions() {
				price, err := s.marketData.GetPrice(stream)
				if err != nil {
					// log failed to generate price
					continue
				}
				tick := Tick{S: stream, P: price}
				for id, conn := range clients {
					log.Debug().Msgf("publishing prices for %s to client %v", stream, id)
					resp, err := json.Marshal(tick)
					if err != nil {
						log.Error().Stack().Err(err).Msg("failed to marshal json")
						continue
					}
					if err := conn.WriteMessage(websocket.TextMessage, resp); err != nil {
						log.Error().Stack().Err(err).Msgf("failed to write message to client %s", id)
						s.manager.RemoveClient(id, nil)
					}
				}
			}
		}
	}
}

func (s *Server) subscribe(c *websocket.Conn, clientID string, streams []string) error {
	if err := s.manager.AddClient(clientID, streams, c); err != nil {
		return fmt.Errorf("Failed to add client %v, err=%w", clientID, err)
	}

	resp := Result{Result: nil, Id: clientID}
	result, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("Failed marshal json for client %v, err=%w", clientID, err)
	}

	if err := c.WriteMessage(websocket.TextMessage, result); err != nil {
		return fmt.Errorf("Failed to write message to client %v, err=%w", clientID, err)
	}
	return nil
}

func (s *Server) unSubscribe(c *websocket.Conn, clientID string, streams []string) error {
	if err := s.manager.RemoveClient(clientID, streams); err != nil {
		return fmt.Errorf("Failed to remove client %v, err=%w", clientID, err)
	}

	resp := Result{Result: nil, Id: clientID}
	result, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("Failed marshal json for client %v, err=%w", clientID, err)
	}

	if err := c.WriteMessage(websocket.TextMessage, result); err != nil {
		return fmt.Errorf("Failed to write message to client %v, err=%w", clientID, err)
	}
	return nil
}
