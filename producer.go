package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"regexp"
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

func NewProducer(uri, exchange, routingKey, contentType string, headers []string) (*Producer, error) {
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
	table, err := parse(headers)
	if err != nil {
		return nil, err
	}
	return &Producer{
		channel:     channel,
		exchange:    exchange,
		routingKey:  routingKey,
		contentType: contentType,
		headers:     table,
		done:        make(chan error),
	}, nil
}

func parse(headers []string) (amqp.Table, error) {
	table := make(amqp.Table)
	for _, header := range headers {
		if err := parse2(header, table); err != nil {
			return nil, err
		}
	}
	return table, nil
}

func parse2(header string, table amqp.Table) error {
	values := strings.Split(header, ":")
	if len(values) == 2 {
		table[values[0]] = parse3(values[1])
		return nil
	}
	return fmt.Errorf("bad %s", header)
}

func parse3(value string) interface{} {
	re := regexp.MustCompile("^\\[(.*)\\]$")
	if match := re.FindStringSubmatch(value); match != nil {
		values := strings.Split(match[1], ",")
		// this terrible dance is required to have the field properly written
		// the lib cases on []interfaces{}, not []string
		//
		// values |> filter \s s==""
		var prunes []string
		for _, v := range values {
			if v != "" {
				prunes = append(prunes, v)
			}
		}
		// https://golang.org/doc/faq#convert_slice_of_interface
		hack := make([]interface{}, len(prunes))
		for i, v := range prunes {
			hack[i] = v
		}
		return hack
	}
	return value
}

func (p *Producer) Publish(body string) error {
	log.Printf("%d bytes to <%s:%s> as %s with headers:%s\n%s", len(body), p.exchange, p.routingKey, p.contentType, p.headers, body)

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
