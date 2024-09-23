package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	DurableSimpleQueue SimpleQueueType = iota
	TransientSimpleQueue
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T),
) error {
	rabbitChan, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}

	msgs, err := rabbitChan.Consume(
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

	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	go func() {
		defer rabbitChan.Close()
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}
			handler(target)
			msg.Ack(false)
		}
	}()
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	rabbitChan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	sQ, err := newSimpleQueue(queueName, simpleQueueType)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	queue, err := rabbitChan.QueueDeclare(
		sQ.name,
		sQ.durable,
		sQ.autoDelete,
		sQ.exclusive,
		sQ.noWait,
		sQ.args,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = rabbitChan.QueueBind(
		sQ.name,
		key,
		exchange,
		sQ.noWait,
		sQ.args,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return rabbitChan, queue, nil
}

type simpleQueue struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table
}

func newSimpleQueue(
	queueName string,
	simpleQueueType SimpleQueueType,
) (*simpleQueue, error) {
	var durable, autoDelete, exclusive bool
	switch simpleQueueType {
	case DurableSimpleQueue:
		durable = true
		autoDelete = false
		exclusive = false
	case TransientSimpleQueue:
		durable = false
		autoDelete = true
		exclusive = true
	default:
		return nil, fmt.Errorf("error creating SimpleQueue, wrong SimpleQueueType")
	}

	var noWait bool
	var args amqp.Table
	noWait = false
	args = nil

	return &simpleQueue{
		queueName, durable, autoDelete, exclusive, noWait, args,
	}, nil
}
