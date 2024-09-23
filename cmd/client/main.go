package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/albsko/learn-pub-sub/internal/gamelogic"
	"github.com/albsko/learn-pub-sub/internal/pubsub"
	"github.com/albsko/learn-pub-sub/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connUrl := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connUrl)
	if err != nil {
		log.Fatalf("Failed dialing %s:%+v", connUrl, err)
	}
	defer conn.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Failed to retrieve username: %+v", err)
	}

	exchange := routing.ExchangePerilDirect
	key := routing.PauseKey
	queueName := fmt.Sprintf("%s.%s", key, username)

	simpleQueue, err := pubsub.NewSimpleQueue(
		queueName,
		pubsub.TransientSimpleQueueType,
	)
	if err != nil {
		log.Fatalf("Failed to create simpleQueue: %+v", err)
	}

	channel, queue, err := pubsub.DeclareAndBind(
		conn,
		exchange,
		key,
		simpleQueue,
	)
	if err != nil {
		log.Fatalf("Failed to declare and bind queue: %+v", err)
	}
	defer channel.Close()

	fmt.Printf("Queue declared: %+v\n", queue)

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	sig := <-signalChan

	fmt.Printf("\nReceived signal (%v). Shutting down RabbitMQ client...\n", sig)
}
