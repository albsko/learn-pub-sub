package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

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
		log.Fatalf("failed dialing %s:%+v", connUrl, err)
	}
	defer conn.Close()

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed creating channel: %+v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("failed to retrieve username: %+v", err)
	}

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.TransientSimpleQueue,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.DurableSimpleQueue,
		handlerWar(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.TransientSimpleQueue,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

LOOP:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Println(err)
				continue
			}

			err = pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+mv.Player.Username,
				mv,
			)
			if err != nil {
				log.Printf("error: %+v\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "spawn":
			err = gs.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) < 2 {
				fmt.Println("usage: spam <n>")
				continue
			}
			n, err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Printf("error: %s is not a valid number\n", words[1])
				continue
			}
			for i := 0; i < n; i++ {
				msg := gamelogic.GetMaliciousLog()
				err = publishGameLog(publishCh, username, msg)
				if err != nil {
					fmt.Printf("error publishing malicious log: %s\n", err)
				}
			}
			fmt.Printf("Published %v malicious logs\n", n)
		case "quit":
			gamelogic.PrintQuit()
			break LOOP
		default:
			fmt.Println("unknown command")
		}
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	sig := <-signalChan

	fmt.Printf("\nReceived signal (%v). Shutting down RabbitMQ client...\n", sig)
}

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}
