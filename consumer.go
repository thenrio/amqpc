package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

type Consumer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	tag        string
	done       chan error
}

func NewConsumer(amqpURI, queue, ctag string) (*Consumer, error) {
	c := &Consumer{
		connection: nil,
		channel:    nil,
		tag:        ctag,
		done:       make(chan error),
	}

	var err error

	log.Printf("Connecting to %s", amqpURI)
	c.connection, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	log.Printf("Getting Channel")
	c.channel, err = c.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("hannel: %s", err)
	}

	log.Printf("starting Consume (consumer tag '%s')", c.tag)
	deliveries, err := c.channel.Consume(
		queue, // name
		c.tag, // consumerTag,
		true,  // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Consume: %s", err)
	}

	go handle(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := c.connection.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	defer log.Printf("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		log.Printf(
			"Got %dB delivery: [%v] %s",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)
	}
	log.Printf("Handle: deliveries channel closed")
	done <- nil
}
