package pubsub

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	key string,
	simpleQueue *SimpleQueue,
) (*amqp.Channel, amqp.Queue, error) {
	rabbitChan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	queue, err := rabbitChan.QueueDeclare(
		simpleQueue.name,
		simpleQueue.durable,
		simpleQueue.autoDelete,
		simpleQueue.exclusive,
		simpleQueue.noWait,
		simpleQueue.args,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = rabbitChan.QueueBind(
		simpleQueue.name,
		key,
		exchange,
		simpleQueue.noWait,
		simpleQueue.args,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return rabbitChan, queue, nil
}

const (
	DurableSimpleQueueType = iota
	TransientSimpleQueueType
)

type SimpleQueue struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table
}

func NewSimpleQueue(
	queueName string,
	simpleQueueType int, // DurableSimpleQueueType or TransientSimpleQueueType
) (*SimpleQueue, error) {
	var durable, autoDelete, exclusive bool
	switch simpleQueueType {
	case DurableSimpleQueueType:
		durable = true
		autoDelete = false
		exclusive = false
	case TransientSimpleQueueType:
		durable = false
		autoDelete = true
		exclusive = true
	default:
		return nil, fmt.Errorf("error creating SimpleQueue, wrong simpleQueueType")
	}

	var noWait bool
	var args amqp.Table
	noWait = false
	args = nil

	return &SimpleQueue{
		queueName, durable, autoDelete, exclusive, noWait, args,
	}, nil
}
