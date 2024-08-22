package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type SubscriptionRequest struct {
	Method  string   `json:"method"`
	Streams []string `json:"streams"`
	Id      string   `json:"id"`
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/prices"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	sub := SubscriptionRequest{
		Method:  "SUBSCRIBE",
		Streams: []string{"BTC"},
		Id:      "1",
	}
	res, err := json.Marshal(sub)
	if err != nil {
		log.Println("failed to marshal json", err)
	}

	log.Println(string(res))

	if err := c.WriteMessage(websocket.TextMessage, res); err != nil {
		log.Println("failed to write message", err)
	}

	unsubtimer := time.After(5 * time.Minute)
	for {
		select {
		case <-done:
			return
		case <-unsubtimer:
			sub := SubscriptionRequest{
				Method:  "UNSUBSCRIBE",
				Streams: []string{"BTC"},
				Id:      "1",
			}
			res, err := json.Marshal(sub)
			if err != nil {
				log.Println("failed to marshal json", err)
			}

			if err := c.WriteMessage(websocket.TextMessage, res); err != nil {
				log.Println("failed to write message", err)
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}