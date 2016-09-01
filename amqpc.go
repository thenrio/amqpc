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
	DEFAULT_CONCURRENCY        int  = 1
	DEFAULT_CONCURRENCY_PERIOD int  = 0
	DEFAULT_QUIET              bool = false
)

var silent bool

// Flags
var (
	// RabbitMQ related
	uri      = flag.String("u", "amqp://guest:guest@localhost:5672", "AMQP URI")
	exchange = flag.String("e", "", "exchange on which to pub")

	// Test bench related
	concurrency       = flag.Int("g", DEFAULT_CONCURRENCY, "Concurrency")
	concurrencyPeriod = flag.Int("gp", DEFAULT_CONCURRENCY_PERIOD, "Concurrency period in ms (Producer only) - Interval at which spawn new Producer when concurrency is set")
	interval          = flag.Int("i", 0, "Interval at which send messages (in ms)")
	messageCount      = flag.Int("n", 1, "Number of messages to send, use 0 for infinite loop")

	// message
	header      = flag.String("header", "", "optional header, value is k:v much like curl ( multiple value behavior unknown )")
	contentType = flag.String("content-type", "application/octet-stream", "content-type is not in headers amqp protocol...")
)

func init() {
	flag.BoolVar(&silent, "silent", false, "silent ( mute )")
	flag.BoolVar(&silent, "s", false, "shorthand for silent")

	flag.Usage = func() {
		fmt.Printf(`
  amqpc [options] routingkey < file

  file is processed using text.template with one argument, the index of message
  index starts at 1

  eg: publish 10 messages to default exchange ( '' ), routing key central.events, each having id in sequence ( 1..10 )

    echo '{"id":{{ . }}}' | amqpc -n=10 central.events

  eg: pub 1 message to somewhere with content-type:application/vnd.me.awesome.1+json

  echo 'message nº{{ . }}' | amqpc -n=1 --content-type=application/vnd.me.awesome.1+json somewhere

	eg: pub 1 message to somewhere with header include:[batteries]

	echo 'message nº{{ . }}' | amqpc -n=1 --header=include:[batteries] somewhere


  see
  * http://golang.org/pkg/text/template/
  * https://golang.org/pkg/fmt/

`)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if silent {
		log.SetOutput(ioutil.Discard)
	}
}

func main() {
	done := make(chan error)

	args := flag.Args()

	routingKey := args[0]
	bytes, _ := ioutil.ReadAll(os.Stdin)
	body := string(bytes[:])
	for i := 0; i < *concurrency; i++ {
		if *concurrencyPeriod > 0 {
			time.Sleep(time.Duration(*concurrencyPeriod) * time.Millisecond)
		}
		go startProducer(done, *uri, *exchange, routingKey, *messageCount, *interval, *contentType, *header, body)
	}

	err := <-done
	if err != nil {
		log.Fatalf("Error : %s", err)
	}

	log.Printf("Exiting...")
}

func startProducer(done chan error, uri string, exchange string, routingKey string, messageCount, interval int, contentType string, header string, body string) {
	var (
		p   *Producer = nil
		err error     = nil
	)

	if err != nil {
		log.Fatalf("Error while starting producer : %s", err)
	}

	for {
		p, err = NewProducer(uri, exchange, routingKey, contentType, header)
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
		p.Publish(_body(template, i))

		i++
		if messageCount != 0 && i > messageCount {
			break
		}

		time.Sleep(duration)
	}

	done <- nil
}

func _body(template *template.Template, i int) string {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, i); err != nil {
		panic(err)
	}
	return buffer.String()
}
