package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"toy.com/websocket/internal/health"
	"toy.com/websocket/internal/stream"
)

var (
	PUBLISHFREQ time.Duration = 10 * time.Second
	PINGFREQ    time.Duration = 1 * time.Minute
	PONGWAIT    time.Duration = 2 * time.Minute
	STREAMS     []string      = []string{"BTC", "ETH", "SOL"}
)

func main() {
	marketData := stream.NewPriceGenerator()
	server := stream.NewServer(marketData, STREAMS, PINGFREQ, PONGWAIT)

	streamHandler := stream.NewStreamHandler(server)
	healthHandler := health.NewHealthHandler()

	go server.PublishPrices(PUBLISHFREQ)

	http.Handle("/prices", streamHandler)
	http.Handle("/health", healthHandler)

	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal().Err(err).Msg("failed to start the server")
	}
}
