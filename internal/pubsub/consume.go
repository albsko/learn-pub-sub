package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int
type AckType int

const (
	DurableSimpleQueue SimpleQueueType = iota
	TransientSimpleQueue
)

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	rabbitCh, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}

	msgs, err := rabbitCh.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	go func() {
		defer rabbitCh.Close()
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			switch handler(target) {
			case Ack:
				msg.Ack(false)
				fmt.Println("Ack")
			case NackDiscard:
				msg.Nack(false, false)
				fmt.Println("NackDiscard")
			case NackRequeue:
				msg.Nack(false, true)
				fmt.Println("NackRequeue")
			}
		}
	}()
	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshaller)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		var target T
		decoder := gob.NewDecoder(bytes.NewReader(data))
		err := decoder.Decode(&target)
		return target, err
	}

	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshaller)
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	rabbitCh, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	queue, err := rabbitCh.QueueDeclare(
		queueName,                             // name
		simpleQueueType == DurableSimpleQueue, // durable
		simpleQueueType != DurableSimpleQueue, // delete when unused
		simpleQueueType != DurableSimpleQueue, // exclusive
		false,                                 // no-wait
		args,                                  // args
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = rabbitCh.QueueBind(
		queue.Name, // queue name
		key,        // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}
	return rabbitCh, queue, nil
}
