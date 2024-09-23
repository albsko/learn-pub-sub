package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/albsko/learn-pub-sub/internal/pubsub"
	"github.com/albsko/learn-pub-sub/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connUrl := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connUrl)
	if err != nil {
		log.Fatalf("Failed dialing %s:%+v", connUrl, err)
	}
	defer conn.Close()

	fmt.Printf("Connected to: %s\n", connUrl)

	rabbitChan, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed creating RabbitMQ Channel: %+v", err)
	}
	defer rabbitChan.Close()

	err = pubsub.PublishJSON[routing.PlayingState](
		rabbitChan,
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

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	sig := <-signalChan

	fmt.Printf("\nReceived signal (%v). Shutting down RabbitMQ connection...\n", sig)
}
