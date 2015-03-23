package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

type Producer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	tag        string
	done       chan error
}

func NewProducer(amqpURI, exchange, exchangeType, key, ctag string, reliable bool) (*Producer, error) {
	p := &Producer{
		connection: nil,
		channel:    nil,
		tag:        ctag,
		done:       make(chan error),
	}

	var err error

	log.Printf("Connecting to %s", amqpURI)
	p.connection, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %v", err)
	}

	log.Printf("Getting Channel ")
	p.channel, err = p.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %v", err)
	}
	return p, nil
}

func (p *Producer) Publish(exchange, routingKey, body string) error {
	log.Printf("Publishing %d bytes to <%s:%s> \n%s", len(body), exchange, routingKey, body)

	if err := p.channel.Publish(
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            []byte(body),
			DeliveryMode:    2, // 1=non-persistent, 2=persistent
			Priority:        0, // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %v", err)
	}

	return nil
}
