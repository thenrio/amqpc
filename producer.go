package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"strings"
)

type Producer struct {
	channel     *amqp.Channel
	exchange    string
	routingKey  string
	contentType string
	headers     amqp.Table
	done        chan error
}

func NewProducer(uri, exchange, routingKey, contentType, header string) (*Producer, error) {
	log.Printf("Connecting to %s", uri)
	connection, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("Dial: %v", err)
	}

	log.Printf("Getting Channel ")
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %v", err)
	}
	headers, err := parse(header)
	if err != nil {
		return nil, err
	}
	return &Producer{
		channel:     channel,
		exchange:    exchange,
		routingKey:  routingKey,
		contentType: contentType,
		headers:     headers,
		done:        make(chan error),
	}, nil
}

func parse(header string) (amqp.Table, error) {
	if len(header) == 0 {
		return amqp.Table{}, nil
	}
	values := strings.Split(header, ":")
	if len(values) == 2 {
		return amqp.Table{values[0]: values[1]}, nil
	}
	return nil, fmt.Errorf("bad %s", header)
}

func (p *Producer) Publish(body string) error {
	log.Printf("Publishing %d bytes to <%s:%s> \n%s", len(body), p.exchange, p.routingKey, body)

	if err := p.channel.Publish(
		p.exchange,   // publish to an exchange
		p.routingKey, // routing to 0 or more queues
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Headers:         p.headers,
			ContentType:     p.contentType,
			ContentEncoding: "",
			Body:            []byte(body),
			DeliveryMode:    2, // 1=non-persistent, 2=persistent
			Priority:        0, // 0-9
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %v", err)
	}

	return nil
}
