package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/albsko/learn-pub-sub/internal/gamelogic"
	"github.com/albsko/learn-pub-sub/internal/pubsub"
	"github.com/albsko/learn-pub-sub/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	connUrl := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connUrl)
	if err != nil {
		log.Fatalf("Failed dialing %s:%+v", connUrl, err)
	}
	defer conn.Close()

	fmt.Printf("Connected to: %s\n", connUrl)

	rabbitCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed creating RabbitMQ Channel: %+v", err)
	}
	defer rabbitCh.Close()

	err = pubsub.PublishJSON(
		rabbitCh,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: true,
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish message: %+v", err)
	}

	fmt.Println("Message published successfully")

	_, queue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.DurableSimpleQueue,
	)
	if err != nil {
		log.Fatalf("Failed to declare and bind game_logs queue: %+v", err)
	}

	fmt.Printf("Game logs queue declared: %+v\n", queue)

	gamelogic.PrintServerHelp()

LOOP:
	for {
		words := gamelogic.GetInput()

		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			log.Println("Server is sending pause message")
			err = pubsub.PublishJSON(
				rabbitCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			if err != nil {
				log.Printf("Failed to publish pause message: %+v", err)
			}
		case "resume":
			log.Println("Server is sending resume message")
			err = pubsub.PublishJSON(
				rabbitCh,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			if err != nil {
				log.Printf("Failed to publish resume message: %+v", err)
			}
		case "quit":
			log.Println("Exiting...")
			break LOOP
		default:
			log.Printf("Don't understand the command: %s", strings.Join(words, " "))
		}
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	sig := <-signalChan

	fmt.Printf("\nReceived signal (%v). Shutting down RabbitMQ server...\n", sig)
}
