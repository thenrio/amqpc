package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
	"time"
)

const (
	DEFAULT_EXCHANGE_TYPE      string = "direct"
	DEFAULT_QUEUE              string = "amqpc-queue"
	DEFAULT_ROUTING_KEY        string = "amqpc-key"
	DEFAULT_CONSUMER_TAG       string = "amqpc-consumer"
	DEFAULT_RELIABLE           bool   = true
	DEFAULT_INTERVAL           int    = 500
	DEFAULT_MESSAGE_COUNT      int    = 0
	DEFAULT_CONCURRENCY        int    = 1
	DEFAULT_CONCURRENCY_PERIOD int    = 0
	DEFAULT_QUIET              bool   = false
)

var (
	exchange   *string
	routingKey *string
	queue      *string
	body       *string
)

// Flags
var (
	consumer = flag.Bool("c", true, "Act as a consumer")
	producer = flag.Bool("p", false, "Act as a producer")

	// RabbitMQ related
	uri          = flag.String("u", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchangeType = flag.String("t", DEFAULT_EXCHANGE_TYPE, "Exchange type - direct|fanout|topic|x-custom")
	consumerTag  = flag.String("ct", DEFAULT_CONSUMER_TAG, "AMQP consumer tag (should not be blank)")
	reliable     = flag.Bool("r", DEFAULT_RELIABLE, "Wait for the publisher confirmation before exiting")
	quiet        = flag.Bool("q", DEFAULT_QUIET, "Turn off output")

	// Test bench related
	concurrency       = flag.Int("g", DEFAULT_CONCURRENCY, "Concurrency")
	concurrencyPeriod = flag.Int("gp", DEFAULT_CONCURRENCY_PERIOD, "Concurrency period in ms (Producer only) - Interval at which spawn new Producer when concurrency is set")
	interval          = flag.Int("i", DEFAULT_INTERVAL, "(Producer only) Interval at which send messages (in ms)")
	messageCount      = flag.Int("n", DEFAULT_MESSAGE_COUNT, "(Producer only) Number of messages to send")
)

func usage() {
	readme := `
  producer
  --------
    amqpc [options] -p exchange routingkey < file

    file is processed using text.template with one argument, the index of message
    index starts at 1
  
    eg: publish messages to default exchange ( '' ), routing key central.events

    echo "message nº{{ . }}" | amqpc -c -n=1 '' central.events

  `
	fmt.Fprintf(os.Stderr, readme)
	flag.PrintDefaults()
	os.Exit(1)
}

func init() {
	flag.Usage = usage
	flag.Parse()
}

func main() {
	done := make(chan error)

	flag.Usage = usage
	args := flag.Args()

	if *quiet {
		log.SetOutput(ioutil.Discard)
	}

	if *producer {
		exchange = &args[0]
		routingKey = &args[1]
		bytes, _ := ioutil.ReadAll(os.Stdin)
		body := string(bytes[:])
		for i := 0; i < *concurrency; i++ {
			if *concurrencyPeriod > 0 {
				time.Sleep(time.Duration(*concurrencyPeriod) * time.Millisecond)
			}
			go startProducer(done, body, *messageCount, *interval)
		}
	} else {
		queue = &args[0]
		for i := 0; i < *concurrency; i++ {
			go startConsumer(done)
		}
	}

	err := <-done
	if err != nil {
		log.Fatalf("Error : %s", err)
	}

	log.Printf("Exiting...")
}

func startConsumer(done chan error) {
	_, err := NewConsumer(
		*uri,
		*queue,
		*consumerTag,
	)

	if err != nil {
		log.Fatalf("Error while starting consumer : %s", err)
	}

	<-done
}

func startProducer(done chan error, body string, messageCount, interval int) {
	var (
		p   *Producer = nil
		err error     = nil
	)

	if err != nil {
		log.Fatalf("Error while starting producer : %s", err)
	}

	for {
		p, err = NewProducer(
			*uri,
			*exchange,
			*exchangeType,
			*routingKey,
			*consumerTag,
			true,
		)
		if err != nil {
			log.Printf("Error while starting producer : %s", err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	i := 1
	duration := time.Duration(interval) * time.Millisecond
	template := template.Must(template.New("body").Parse(body))

	for {
		publish(p, _body(template, i))

		i++
		if messageCount != 0 && i > messageCount {
			break
		}

		time.Sleep(duration)
	}

	done <- nil
}

func _body(template *template.Template, i int) string {
	buffer := new(bytes.Buffer)
	if err := template.Execute(buffer, i); err != nil {
		panic(err)
	}
	return buffer.String()
}

func publish(p *Producer, body string) {
	p.Publish(*exchange, *routingKey, body)
}
